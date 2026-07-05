#!/usr/bin/env ruby
# frozen_string_literal: true

require 'pathname'
require 'yaml'

ROOT = Pathname.new(__dir__).join('..').expand_path
ENTRY = ROOT.join('contracts/openapi.yaml')

# 这里只做本地拆分文件的 YAML/$ref 完整性检查；完整 OpenAPI 规范校验
# 以后可以交给 Redocly CLI 或 OpenAPI Generator 这类专用工具。
class RefValidator
  def initialize(root)
    @root = root
    @cache = {}
    @checked_refs = 0
  end

  attr_reader :checked_refs

  def validate_file(path)
    document = load_yaml(path)
    each_ref(document) do |ref|
      resolve_ref(path, ref)
      @checked_refs += 1
    end
  end

  private

  def load_yaml(path)
    absolute = path.expand_path
    @cache[absolute.to_s] ||= YAML.load_file(absolute.to_s)
  rescue Errno::ENOENT
    abort "OpenAPI reference target does not exist: #{relative(absolute)}"
  rescue Psych::SyntaxError => e
    abort "OpenAPI YAML syntax error in #{relative(absolute)}: #{e.message}"
  end

  def each_ref(value, &block)
    case value
    when Hash
      value.each do |key, child|
        yield child if key == '$ref' && child.is_a?(String)
        each_ref(child, &block)
      end
    when Array
      value.each { |child| each_ref(child, &block) }
    end
  end

  def resolve_ref(base_path, ref)
    file_part, fragment = ref.split('#', 2)
    target_path = if file_part.nil? || file_part.empty?
                    base_path.expand_path
                  else
                    base_path.dirname.join(file_part).cleanpath.expand_path
                  end

    document = load_yaml(target_path)
    resolve_fragment(target_path, document, fragment)
  end

  def resolve_fragment(target_path, document, fragment)
    return document if fragment.nil? || fragment.empty?

    unless fragment.start_with?('/')
      abort "Unsupported OpenAPI fragment in #{relative(target_path)}: ##{fragment}"
    end

    fragment.split('/')[1..].each do |raw_segment|
      segment = raw_segment.gsub('~1', '/').gsub('~0', '~')
      if document.is_a?(Hash) && document.key?(segment)
        document = document[segment]
      elsif document.is_a?(Array) && segment.match?(/\A\d+\z/) && document[segment.to_i]
        document = document[segment.to_i]
      else
        abort "Broken OpenAPI reference in #{relative(target_path)}: ##{fragment}"
      end
    end

    document
  end

  def relative(path)
    path.relative_path_from(@root)
  rescue ArgumentError
    path
  end
end

files = [ENTRY, *ROOT.glob('contracts/openapi/**/*.yaml')].uniq.sort
validator = RefValidator.new(ROOT)
files.each { |path| validator.validate_file(path) }

puts "OpenAPI references OK: checked #{validator.checked_refs} refs across #{files.length} files."
