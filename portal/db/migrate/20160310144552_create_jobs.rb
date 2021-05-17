class CreateJobs < ActiveRecord::Migration
  def change
    create_table :jobs do |t|
      t.string :status
      t.references :team
      t.timestamps null: false
    end
  end
end
