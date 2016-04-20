class Job < ActiveRecord::Base
  belongs_to :team

  def running?
    status == 'Running'
  end

  def waiting?
    status == 'Waiting'
  end

  def enqueued?
    running? || waiting?
  end
end
