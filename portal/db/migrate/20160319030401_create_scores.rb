class CreateScores < ActiveRecord::Migration[5.1]
  def change
    create_table :scores do |t|
      t.boolean :pass
      t.integer :score
      t.text :message
      t.references :team
      t.timestamps null: false
    end
  end
end
