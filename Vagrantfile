# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/focal64"
  config.vm.network "forwarded_port", guest: 22, host: 2222
   config.vm.provider "virtualbox" do |v|
     v.name = "yarp-node"
     v.memory = "1024"
  end

  ssh_pub_key = File.readlines("#{Dir.home}/.ssh/id_rsa.pub").first.strip
  config.vm.provision "shell", inline: <<-SHELL
     echo #{ssh_pub_key} >> /root/.ssh/authorized_keys
     apt-get update
     apt-get -y install net-tools
   SHELL
end
