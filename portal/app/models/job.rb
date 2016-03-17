class Job < ActiveRecord::Base
  belongs_to :team

  def self.time_wait?(team:)
    where(team: team).where('created_at > ?', Time.current - 60.second).count > 0
  end
end
