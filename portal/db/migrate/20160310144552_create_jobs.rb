class CreateJobs < ActiveRecord::Migration[5.1]
  def change
    create_table :jobs do |t|
      t.string :status
      t.references :team
      t.timestamps null: false
    end
  end
end
