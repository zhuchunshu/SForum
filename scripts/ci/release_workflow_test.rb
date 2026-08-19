#!/usr/bin/env ruby
# frozen_string_literal: true

require "yaml"

root = File.expand_path("../..", __dir__)
release = YAML.load_file(File.join(root, ".github/workflows/release.yml"))
ci = YAML.load_file(File.join(root, ".github/workflows/ci.yml"))

def fail!(message)
  abort "release_workflow_test.rb: #{message}"
end

def needs(job)
  Array(job.fetch("needs", []))
end

release_jobs = release.fetch("jobs")
publish = release_jobs.fetch("publish-candidate-platform")
matrix = publish.dig("strategy", "matrix", "include")
fail!("platform publish matrix must contain six entries") unless matrix.is_a?(Array) && matrix.length == 6

images = %w[sforum-api sforum-migrate sforum-web]
architectures = {
  "amd64" => ["linux/amd64", "ubuntu-latest"],
  "arm64" => ["linux/arm64", "ubuntu-24.04-arm"]
}

images.product(architectures.keys).each do |image, arch|
  entry = matrix.find { |candidate| candidate["image"] == image && candidate["arch"] == arch }
  fail!("missing #{image}/#{arch} native build") unless entry

  platform, runner = architectures.fetch(arch)
  fail!("#{image}/#{arch} has the wrong platform") unless entry["platform"] == platform
  fail!("#{image}/#{arch} has the wrong runner") unless entry["runner"] == runner

  expected_writer = %w[sforum-api sforum-web].include?(image)
  fail!("#{image}/#{arch} has the wrong cache writer policy") unless entry["cache_writer"] == expected_writer
end

release_jobs.each_value do |job|
  Array(job["steps"]).each do |step|
    uses = step["uses"].to_s
    fail!("release workflow must not use QEMU") if uses.include?("docker/setup-qemu-action")
  end
end

publish_steps = publish.fetch("steps")
build_index = publish_steps.index { |step| step["uses"].to_s.include?("docker/build-push-action") }
scan_index = publish_steps.index { |step| step["uses"].to_s.include?("aquasecurity/trivy-action") }
upload_index = publish_steps.index { |step| step["uses"].to_s.include?("actions/upload-artifact") }
fail!("platform build, scan, and digest upload steps are required") unless build_index && scan_index && upload_index
fail!("digest must be uploaded only after its platform scan passes") unless build_index < scan_index && scan_index < upload_index

build_inputs = publish_steps.fetch(build_index).fetch("with")
fail!("platform build must use the matrix platform") unless build_inputs["platforms"] == "${{ matrix.platform }}"
scan_environment = publish_steps.fetch(scan_index).fetch("env", {})
fail!("platform scan must use the matrix platform") unless scan_environment["TRIVY_PLATFORM"] == "${{ matrix.platform }}"
outputs = build_inputs["outputs"].to_s
unless outputs.include?("push-by-digest=true") && outputs.include?("name-canonical=true")
  fail!("platform build must push canonical digest-only candidates")
end

merge = release_jobs.fetch("merge-candidate")
fail!("manifest merge must wait for all platform candidates") unless needs(merge).include?("publish-candidate-platform")
fail!("manifest merge matrix must cover all three images") unless merge.dig("strategy", "matrix", "image") == images
promote = release_jobs.fetch("promote")
fail!("promotion matrix must cover only the three supported images") unless promote.dig("strategy", "matrix", "image") == images

merge_command = merge.fetch("steps").map { |step| step["run"] }.compact.find do |command|
  command.include?("docker buildx imagetools create")
end

github_release = release_jobs.fetch("github-release")
release_command = github_release.fetch("steps").map { |step| step["run"] }.compact.find do |command|
  command.include?("gh release create")
end
fail!("GitHub release creation command is missing") unless release_command
unless release_command.include?("generate-release-notes.sh") &&
       release_command.include?("--notes-file") &&
       !release_command.include?("--generate-notes")
  fail!("GitHub releases must use the tested commit-list notes generator")
end
fail!("manifest merge command is missing") unless merge_command
unless merge_command.include?('sort == ["amd64", "arm64"]') && merge_command.include?("manifest_digest")
  fail!("manifest merge must verify both runtime architectures and export its digest")
end

%w[smoke release-assets].each do |job_name|
  job_needs = needs(release_jobs.fetch(job_name))
  fail!("#{job_name} must wait for merged candidates") unless job_needs.include?("merge-candidate")
  fail!("#{job_name} must not consume unmerged platform candidates") if job_needs.include?("publish-candidate-platform")
end

asset_matrix = release_jobs.fetch("release-assets").dig("strategy", "matrix", "include")
expected_asset_matrix = [
  { "goos" => "linux", "goarch" => "amd64" },
  { "goos" => "linux", "goarch" => "arm64" },
  { "goos" => "darwin", "goarch" => "amd64" },
  { "goos" => "darwin", "goarch" => "arm64" }
]
unless asset_matrix == expected_asset_matrix
  fail!("release assets must target only Linux and macOS on amd64/arm64")
end

ci_jobs = ci.fetch("jobs")
fail!("CI must not keep a duplicate Web dependency build") if ci_jobs.key?("web-runtime-vulnerabilities")

containers = ci_jobs.fetch("containers")
container_needs = needs(containers)
fail!("container builds must start immediately") unless container_needs.empty?

container_steps = containers.fetch("steps")
build_index = container_steps.index do |step|
  step["uses"].to_s.include?("docker/build-push-action")
end
scan_index = container_steps.index do |step|
  step["uses"].to_s.include?("aquasecurity/trivy-action")
end
fail!("CI container build and Web image scan are required") unless build_index && scan_index
fail!("Web image must be scanned after it is built") unless build_index < scan_index

container_build = container_steps.fetch(build_index)
build_inputs = container_build.fetch("with")
unless build_inputs["load"] == "${{ matrix.image == 'web' }}" &&
       build_inputs["tags"] == "${{ matrix.image == 'web' && 'sforum-web:ci' || '' }}"
  fail!("CI must load only the Web image under a stable local tag")
end

scan = container_steps.fetch(scan_index)
fail!("CI image scan must run only for Web") unless scan["if"] == "matrix.image == 'web'"
scan_inputs = scan.fetch("with")
unless scan_inputs["scan-type"] == "image" &&
       scan_inputs["image-ref"] == "sforum-web:ci" &&
       scan_inputs["vuln-type"] == "os,library" &&
       scan_inputs["severity"] == "CRITICAL,HIGH" &&
       scan_inputs["exit-code"] == "1"
  fail!("CI Web scan must block on vulnerabilities in the final runtime image")
end

cache_to = build_inputs["cache-to"].to_s
unless cache_to.include?("github.event_name != 'pull_request'")
  fail!("pull requests must not export max-mode container caches")
end

smoke_script = File.read(File.join(root, "scripts/ci/release-smoke.sh"))
unless smoke_script.include?('docker run --rm "$image" "$binary" --version') &&
       smoke_script.include?('image="${SFORUM_REGISTRY}/sforum-${service}:${SFORUM_VERSION}"')
  fail!("image identity checks must run without Compose service dependencies")
end

puts "release_workflow_test.rb: all checks passed"
