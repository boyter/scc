#!/usr/bin/env packer build --force
#
#  Author: Hari Sekhon
#  Date: 2023-06-13 02:46:59 +0100 (Tue, 13 Jun 2023)
#
#  vim:ts=2:sts=2:sw=2:et:filetype=conf
#
#  https://github.com/HariSekhon/Templates
#
#  License: see accompanying Hari Sekhon LICENSE file
#
#  If you're using my code you're welcome to connect with me on LinkedIn and optionally send me feedback to help steer this or other code I publish
#
#  https://www.linkedin.com/in/HariSekhon
#

# Uses adjacent Debian Preseed from installers/
#
# 'packer' command must be run from the same directory as this file so the preseed.cfg provided is auto-served via HTTP

# ============================================================================ #
#                  P a c k e r   -   F e d o r a   -   Q e m u
# ============================================================================ #

packer {
  # Data sources only available in 1.7+
  required_version = ">= 1.7.0, < 2.0.0"
  required_plugins {
    qemu = {
      version = "~> 1.1"
      source  = "github.com/hashicorp/qemu"
    }
  }
}

# https://alt.fedoraproject.org/alt/
variable "version" {
  type    = string
  default = "38"
}

variable "iso" {
  type    = string
  default = "Fedora-Server-dvd-x86_64-38-1.6.iso"
}

variable "checksum" {
  type    = string
  default = "09dee2cd626a269aefc67b69e63a30bd0baa52d4"
}
locals {
  name    = "fedora"
  url     = "https://download.fedoraproject.org/pub/fedora/linux/releases/${var.version}/Server/x86_64/iso/${var.iso}"
  vm_name = "${local.name}-${var.version}"
  arch    = "x86_64"
}

# https://developer.hashicorp.com/packer/plugins/builders/qemu
source "qemu" "fedora" {
  vm_name              = local.vm_name
  qemu_binary          = "qemu-system-x86_64"
  machine_type         = "pc"
  iso_url              = local.url
  iso_checksum         = var.checksum
  cpus                 = 2
  memory               = 2048
  net_device           = "virtio-net"
  disk_interface       = "virtio-scsi" # or virtio?
  format               = "qcow2"
  disk_discard         = "unmap"
  disk_image           = true
  disk_size            = 40960
  disk_additional_size = []
  output_directory     = "output-${local.vm_name}-${local.arch}"
  headless             = false
  use_default_display  = true # might be needed on Mac to avoid errors about sdl not being available
  http_directory       = "installers"
  ssh_timeout          = "30m"
  ssh_password         = "packer"
  ssh_username         = "packer"
  shutdown_command     = "echo 'packer' | sudo -S shutdown -P now"
  boot_wait            = "5s"
  boot_command = [
    "<up><wait>",
    "e",
    "<down><down><down><left>",
    # leave a space from last arg
    " inst.ks=http://{{.HTTPIP}}:{{.HTTPPort}}/anaconda-ks.cfg <f10>"
  ]
  qemuargs = [
    #["-smbios", "type=1,serial=ds=nocloud-net;instance-id=packer;seedfrom=http://{{ .HTTPIP }}:{{ .HTTPPort }}/"],
    # spice-app isn't respected despite doc https://www.qemu.org/docs/master/system/invocation.html#hxtool-3
    # packer-builder-qemu plugin: Qemu stderr: qemu-system-x86_64: -display spice-app: Parameter 'type' does not accept value 'spice-app'
    #["-display", "spice-app"],
    #["-display", "cocoa"],  # Mac only
    #["-display", "vnc:0"],  # starts VNC by default, but doesn't launch user's vncviewer - ubuntu-x86_64.qemu.pkr.hcl
  ]
  # Only on ARM Macs
  #machine_type = "virt"  # packer-builder-qemu plugin: Qemu stderr: qemu-system-x86_64: unsupported machine type
}


build {
  name = local.name

  sources = ["source.qemu.fedora"]

  # https://developer.hashicorp.com/packer/docs/provisioners/shell-local
  #
  #provisioner "shell-local" {
  #  environment_vars = [
  #    "VM_NAME=${local.vm_name}"
  #  ]
  #  script = "./scripts/local_vboxsf.sh"
  #}

  # https://developer.hashicorp.com/packer/docs/provisioners/shell
  #
  provisioner "shell" {
    scripts = [
      "./scripts/version.sh",
      #"./scripts/mount_vboxsf.sh",
      #"./scripts/collect_anaconda.sh",
      "./scripts/final.sh",
    ]
    execute_command = "{{ .Vars }} echo 'packer' | sudo -S -E bash '{{ .Path }}' '${packer.version}'"
  }

  post-processor "checksum" {
    checksum_types      = ["md5", "sha512"]
    keep_input_artifact = true
    output              = "output-{{.BuildName}}/{{.BuildName}}.{{.ChecksumType}}"
  }
}
