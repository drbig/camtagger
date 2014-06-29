#!/usr/bin/env ruby
# encoding: utf-8
#
# See LICENSE.txt for licensing information.

require 'json'
require 'set'

def usage
  puts "Usage: #{$0} add|del tag,tag... -- file file..."
  exit(1)
end

usage if ARGV.length < 4
usage if ARGV[2] != '--'

MODE = ARGV.shift.downcase
usage unless %w{add del}.member? MODE

tags = ARGV.shift.split(',')
usage if tags.empty?
TAGS = Set.new(tags)
ARGV.shift # remove the '--'

CMDS = {
  :search => {
    :string => 'camtool search -',
    :mode => :pipe,
    :out => :json,
  },
  :describe => {
    :string => 'camtool describe %s',
    :mode => :simple,
    :out => :json,
  },
  :attr_mod => {
    :string => 'camput attr %s %s %s %s',
    :mode => :simple,
    :out => :blob,
  },
}

def ask(command, query)
  cmd = CMDS[command]
  res = nil

  case cmd[:mode]
  when :pipe
    res = IO.popen(cmd[:string], 'r+') do |p|
      p.puts JSON.generate(query)
      p.close_write
      p.read
    end
  when :simple
    res = `#{cmd[:string] % query}`
  end

  case cmd[:out]
  when :json
    JSON.parse(res, :symbolize_names => true)
  when :blob
    res
  end
end

def find_blobs(name, size)
  query = {
    :file => {
      :filename => {
        :equals => name,
      },
      :filesize => {
        :min => size,
        :max => size,
      },
    },
  }
  res = ask(:search, query)

  return nil unless res[:blobs]
  res[:blobs].collect {|e| e[:blob] }
end

def find_permanodes(blob)
  query = {
    :permanode => {
      :attr => 'camliContent',
      :value => blob,
    },
  }
  res = ask(:search, query)

  return nil unless res[:blobs]
  res[:blobs].collect {|e| e[:blob] }
end

def get_attrs(node)
  res = ask(:describe, node)

  return nil if res[:meta].empty?
  res[:meta][node.to_sym][:permanode][:attr]
end

ARGV.each do |arg|
  print arg, ' '

  unless File.exists? arg
    puts 'NON-EXISTENT'
    next
  end

  if File.directory? arg
    puts 'DIRECTORY'
    next
  end

  name = File.basename(arg)
  size = File.size(arg)
  nodes = find_blobs(name, size).compact \
    .collect{|b| find_permanodes(b) }.compact.flatten

  if nodes.length != 1
    puts "#{nodes.length} NODES"
    next
  end

  permanode = nodes.first
  print permanode, ' '

  attrs = get_attrs(permanode)
  unless attrs
    puts 'NO ATTRS'
    next
  end

  if attrs.has_key? :tag
    tags = Set.new(attrs[:tag])
  else
    tags = Set.new
  end

  if MODE == 'add'
    tags = TAGS - tags
  else
    tags = TAGS.intersection(tags)
  end

  if tags.empty?
    puts MODE == 'add' ? 'ALL TAGS PRESENT' : 'NO TAGS TO REMOVE'
  else
    tags.each do |tag|
      print tag, ' '
      res = ask(:attr_mod, ["--#{MODE}", permanode, 'tag', tag])
    end

    puts
  end
end
