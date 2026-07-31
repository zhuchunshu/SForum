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
fail!("platform publish matrix must contain eight entries") unless matrix.is_a?(Array) && matrix.length == 8

images = %w[sforum-api sforum-worker sforum-migrate sforum-web]
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
outputs = build_inputs["outputs"].to_s
unless outputs.include?("push-by-digest=true") && outputs.include?("name-canonical=true")
  fail!("platform build must push canonical digest-only candidates")
end

merge = release_jobs.fetch("merge-candidate")
fail!("manifest merge must wait for all platform candidates") unless needs(merge).include?("publish-candidate-platform")
fail!("manifest merge matrix must cover all four images") unless merge.dig("strategy", "matrix", "image") == images

merge_command = merge.fetch("steps").map { |step| step["run"] }.compact.find do |command|
  command.include?("docker buildx imagetools create")
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

ci_jobs = ci.fetch("jobs")
vulnerability_gate = ci_jobs.fetch("web-runtime-vulnerabilities")
scan = vulnerability_gate.fetch("steps").find do |step|
  step["uses"].to_s.include?("aquasecurity/trivy-action")
end
fail!("CI Web vulnerability scan is missing") unless scan
scan_inputs = scan.fetch("with")
unless scan_inputs["scan-type"] == "fs" &&
       scan_inputs["scan-ref"] == "apps/web/node_modules" &&
       scan_inputs["severity"] == "CRITICAL,HIGH" &&
       scan_inputs["exit-code"] == "1"
  fail!("CI Web vulnerability scan is not blocking the production dependency tree")
end

containers = ci_jobs.fetch("containers")
container_needs = needs(containers)
fail!("container builds must wait for the fast vulnerability gate") unless container_needs.include?("web-runtime-vulnerabilities")
fail!("container builds must run in parallel with the quality gate") if container_needs.include?("quality")

container_build = containers.fetch("steps").find do |step|
  step["uses"].to_s.include?("docker/build-push-action")
end
cache_to = container_build&.dig("with", "cache-to").to_s
unless cache_to.include?("github.event_name != 'pull_request'")
  fail!("pull requests must not export max-mode container caches")
end

puts "release_workflow_test.rb: all checks passed"
