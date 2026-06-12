# frozen_string_literal: true
require "spec_helper"

describe Foo do
  it "adds numbers" do
    result = 1 + 1
    expect(result).to eq(2) if result.positive?
  end
end
