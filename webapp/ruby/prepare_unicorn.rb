# frozen_string_literal: true

require "fileutils"

# The unreleased git revision is needed for Ruby 4, but git installs do not
# include the generated files that are present in released gems.
spec = Gem.loaded_specs["unicorn"]
abort("unicorn gem not found") unless spec

dir = spec.full_gem_path

version_path = File.join(dir, "lib", "unicorn", "version.rb")
FileUtils.mkdir_p(File.dirname(version_path))
File.write(version_path, <<~RUBY)
  # frozen_string_literal: true
  module Unicorn
    module Const
      UNICORN_VERSION = "#{spec.version}"
    end
    VERSION = Const::UNICORN_VERSION
  end
RUBY

ext_dir = File.join(dir, "ext", "unicorn_http")
Dir.chdir(ext_dir) do
  system("ragel", "-G2", "unicorn_http.rl", "-o", "unicorn_http.c") || abort("ragel step failed")
  system(Gem.ruby, "extconf.rb") || abort("extconf.rb failed")
  system("make", "clean")
  system("make") || abort("make failed")
  system("make", "install") || abort("make install failed")
end

shared_object = Dir[File.join(ext_dir, "unicorn_http.so"), File.join(dir, "lib", "unicorn_http.so")].first
FileUtils.cp(shared_object, File.join(dir, "lib", "unicorn_http.so")) if shared_object
