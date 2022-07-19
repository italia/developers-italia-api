#!/usr/bin/env ruby

require 'date'
require 'yaml'
require 'securerandom'

software = []
software_urls = []

time = DateTime.new 2014, 5, 1

1.upto 30 do |i|
  s = {
    'id' => SecureRandom.uuid,
    'publiccode_yml' => '-',
    'created_at' => time.rfc3339,
    'updated_at' => time.rfc3339
  }
  software_urls << {
    'id' => SecureRandom.uuid,
    'software_id' => s['id'],
    'url' => "https://#{i}-a.example.org/code/repo",
    'created_at' => time.rfc3339,
    'updated_at' => time.rfc3339
  }
  software_urls << {
    'id' => SecureRandom.uuid,
    'software_id' => s['id'],
    'url' => "https://#{i}-b.example.org/code/repo",
    'created_at' => time.rfc3339,
    'updated_at' => time.rfc3339
  }

  time += 15

  software << s
end

puts software.to_yaml
File.write("software.yml", software.to_yaml)
File.write("software_urls.yml", software_urls.to_yaml)
