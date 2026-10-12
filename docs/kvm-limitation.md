# Отчёт об ограничении KVM/WSL

## Статус проверки

KVM в WSL на проверенной конфигурации не удалось использовать: Windows не сообщала, что возможности аппаратной виртуализации доступны гостевой среде, в WSL не обнаруживался `/dev/kvm`, а в `/proc/cpuinfo` не находились флаги `vmx`/`svm`. Это фиксирует результат эксперимента, но само по себе не доказывает, что процессор физически не поддерживает виртуализацию.

В ходе проверки использовались Windows 10 Pro и WSL 2. В диагностике на системе с Intel Core i5-10210U PowerShell выводил `VirtualizationFirmwareEnabled=False`, `VMMonitorModeExtensions=False`, `SecondLevelAddressTranslationExtensions=False`. На другой проверенной машине с Intel Celeron N4500 наблюдалось такое же отсутствие признаков виртуализации в Windows. Если в окончательный отчёт включается только одна машина, оставьте данные именно той машины, которую проверяли, и приложите её собственный вывод команд.

## Команды, которыми проверялось ограничение

### В Windows PowerShell

```powershell
get-ciminstance win32_processor | select-object name, virtualizationfirmwareenabled, vmmonitorextensions, secondleveladdresstranslationextensions
```

Для точного вывода свойств используйте надёжный вариант с форматированием списка:

```powershell
get-ciminstance win32_processor | format-list name, virtualizationfirmwareenabled, vmmonitorextensions, secondleveladdresstranslationextensions
```

Дополнительная системная проверка:

```powershell
systeminfo
```

В выводе следует сохранить строки о Hyper-V / требованиях виртуализации, если они присутствуют.

### В Ubuntu под WSL

```bash
uname -m
```

Ожидаемая архитектура — `x86_64`.

```bash
grep -m1 -oE '\b(vmx|svm)\b' /proc/cpuinfo
```

При проверке флаги не выводились.

```bash
ls -l /dev/kvm
```

При проверке файл `/dev/kvm` отсутствовал.

Эти результаты вместе показывают, что KVM недоступен внутри этой WSL-среды. Для корректного вывода о причине нужно отличать отсутствие проброса виртуализации в WSL от физической неподдержки процессором.

## Что должно быть выполнено на подходящем Linux-хосте

На физическом Linux-хосте или Linux VM, которому действительно переданы расширения аппаратной виртуализации, следует выполнить:

```bash
sudo apt update
sudo apt install -y qemu-kvm libvirt-daemon-system libvirt-clients virtinst cpu-checker
```

Затем проверить доступность KVM:

```bash
sudo kvm-ok
```

```bash
ls -l /dev/kvm
```

```bash
sudo virt-host-validate qemu
```

Для настоящего запуска VM эти проверки должны подтвердить, что KVM доступен. Если проверка проводится внутри WSL без проброса аппаратной виртуализации, успешная установка пакетов сама по себе не означает работоспособность KVM.

## Пример команды создания VM для подходящего хоста

Сначала поместите установочный ISO Ubuntu Server 24.04 в `/var/lib/libvirt/images/ubuntu-24.04-live-server-amd64.iso`. Затем выполните на Linux-хосте:

```bash
sudo virt-install --name weatherwise-vm --memory 3072 --vcpus 2 --cpu host --disk path=/var/lib/libvirt/images/weatherwise-vm.qcow2,size=25,format=qcow2 --cdrom /var/lib/libvirt/images/ubuntu-24.04-live-server-amd64.iso --os-variant ubuntu24.04 --network network=default,model=virtio --graphics spice --noautoconsole
```

Для установки с графической консоли:

```bash
virt-viewer weatherwise-vm
```

Параметры примера: 3 ГБ RAM, 2 vCPU, диск 25 ГБ, сеть libvirt NAT. Перед выполнением нужно убедиться, что ISO расположен по указанному пути, установлен графический клиент `virt-viewer`, а хост поддерживает аппаратную виртуализацию и KVM.

После установки Ubuntu в VM проверяются SSH, Docker, Compose и доступность frontend/backend. Если требуется доступ с другой машины, нужно настроить сетевую достижимость гостя либо подходящее перенаправление портов.

## Интерпретация для сдачи

- Docker Compose-приложение было реализовано отдельно и проверено на ноутбуке.
- KVM в текущей WSL-среде фактически не запущен.
- Команда `virt-install` и проверки выше являются планом проверки на подходящем Linux-хосте, а не свидетельством созданной KVM VM.
- Ansible inventory и playbook приложены отдельно как артефакты предполагаемого автоматизированного развёртывания; успешный результат их реального запуска можно заявлять только после выполнения на доступной по SSH Linux VM.
