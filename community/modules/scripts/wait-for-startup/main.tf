/**
 * Copyright 2022 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

locals {
  joined_instance_names = concat(var.instance_names, [var.instance_name])
}

data "google_compute_instance" "vm_instance" {
  count = length(local.joined_instance_names)

  name    = local.joined_instance_names[count.index]
  zone    = var.zone
  project = var.project_id
}

resource "null_resource" "wait_for_startup" {
  provisioner "local-exec" {
    command = "/bin/bash ${path.module}/scripts/wait-for-startup-status.sh"
    environment = {
      INSTANCE_NAMES = join(" ", distinct(compact(data.google_compute_instance.vm_instance[*].name)))
      ZONE           = var.zone
      PROJECT_ID     = var.project_id
      TIMEOUT        = var.timeout
    }
  }
}
