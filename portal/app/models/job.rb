class Job < ActiveRecord::Base
  DEFAULT_TIMEOUT = 60.seconds

  belongs_to :team

  def self.time_wait?(team:)
    where(team: team).where('created_at > ?', Time.current - timeout).count > 0
  end

  private
  def self.timeout
    Rails.application.config.x.benchmarker.timeout || DEFAULT_TIMEOUT
  end
end
