# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/focal64"
  config.vm.box_check_update = false

  config.vm.network "private_network", type: "dhcp"

  config.vm.define "app" do |app|
    app.vm.hostname = "app"
    app.vm.network "forwarded_port", guest: 80, host: 8000
    app.vm.provider "virtualbox" do |vb|
      vb.cpus = 2
      vb.memory = 1500
    end

    app.vm.provision "ansible" do |ansible|
      ansible.verbose = "v"
      ansible.playbook = "./provisioning/image/ansible/playbooks.yml"
      ansible.skip_tags = "nodejs"

      ansible.groups = {
        "guests"  => ["app"]
      }
    end

    app.vm.synced_folder "./", "/vagrant"
  end

  config.vm.define "bench" do |bench|
    bench.vm.hostname = "bench"
    bench.vm.provider "virtualbox" do |vb|
      vb.cpus = 4
      vb.memory = 7680
    end

    bench.vm.provision "ansible" do |ansible|
      ansible.verbose = "v"
      ansible.playbook = "./provisioning/bench/ansible/playbooks.yml"

      ansible.groups = {
        "bench" => ["bench"]
      }
    end
  end
end
