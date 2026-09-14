# SPDX-License-Identifier: MIT

=begin
A block comment, which Ruby spells with =begin and =end in the first column and
nowhere else.
=end

# frozen_string_literal: true

class Holder
  attr_reader :name

  def initialize(name)
    @name = name
    @counts = Hash.new(0)
  end

  def tally(words)
    total = 0
    words.each do |word|
      next if word.empty?

      @counts[word] += 1
      total += 1
    end

    total
  end

  def describe
    single = 'a single quoted string with a " in it'
    double = "a double quoted string with a ' in it and #{name}"
    percent = %w[one two three]

    "#{single} #{double} #{percent.join(' ')}"
  end
end

holder = Holder.new('sample')
total = holder.tally(['one', 'two', '', 'two'])
case total
when 0 then puts 'nothing'
when 1..2 then puts 'some'
else puts 'many'
end

begin
  puts holder.describe
rescue StandardError
  puts 'failed'
ensure
  puts 'done'
end
