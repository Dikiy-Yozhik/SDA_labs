# Дополнение к README WeatherWise: KVM fallback и Ansible

## Текущее приложение

Frontend и backend WeatherWise запускаются через Docker Compose из корня репозитория. Запусти их способом, который уже проверен в основной части лабораторной, и проверь доступность страницы с ноутбука. В этом дополнении не утверждается, что приложение было развёрнуто на KVM VM: KVM в проверенной WSL-среде не оказался доступен.

## Схема предполагаемой автоматизации

```text
ноутбук / linux-контроллер (ansible)
        |
        | ssh
        v
ubuntu server vm (kvm/libvirt)
        |
        | docker compose
        v
frontend container <---- запросы ----> backend container
```

## Почему KVM не запущен

Смотри [`docs/kvm-limitation.md`](docs/kvm-limitation.md): там перечислены команды Windows и WSL, зафиксированные наблюдения, а также пример проверок и создания VM на подходящем Linux-хосте. В отчёте важно указать точную машину, на которой сделаны измерения, и приложить её реальный вывод команд.

## Ansible

Артефакты находятся в `ansible/inventory.ini` и `ansible/deploy.yml`. Перед реальным запуском замените образцовый IP `192.168.122.100` в inventory на IP целевой Ubuntu VM. В контроллере установите Ansible и коллекцию:

```bash
sudo apt update
sudo apt install -y ansible openssh-client
ansible-galaxy collection install community.docker
```

Затем настройте SSH-ключ и выполните:

```bash
ansible -i ansible/inventory.ini weather_server -m ansible.builtin.ping
```

```bash
ansible-playbook -i ansible/inventory.ini ansible/deploy.yml --ask-become-pass
```

Повторите вторую команду без изменения конфигурации, чтобы проверить идемпотентность. Ожидается `failed=0`, `unreachable=0`, а при неизменном состоянии повторный запуск должен в идеале дать `changed=0`. До фактического запуска результаты обозначаются как ожидаемые, а не как уже полученные.

Playbook предполагает наличие в корне проекта `compose.yaml`, `frontend/` и `backend/`. Если Compose использует дополнительные файлы (например, `.env`, который не содержит секретов) или нестандартные каталоги, адаптируйте задачи копирования.

Подробная процедура проверки находится в [`docs/ansible-validation.md`](docs/ansible-validation.md).

## Trivy и GitHub Actions

Workflow `.github/workflows/trivy.yml` собирает образы Compose и сканирует их Trivy. Override `.github/compose.trivy.yaml` предполагает, что Compose-сервисы называются `frontend` и `backend`. Если в `compose.yaml` используются другие имена сервисов, измените имена в override. Результат сканирования появляется в GitHub → Actions; `exit-code: '0'` оставляет workflow успешным и позволяет просмотреть найденные уязвимости, не блокируя учебный pipeline.

## Ограничения и честное представление результата

На текущем компьютере приложение через Compose — практическая часть. KVM/libvirt VM и фактический запуск Ansible нельзя объявлять выполненными, пока нет доступного KVM-хоста и журналов успешного запуска. Playbook, inventory и команды здесь являются подготовленными артефактами для теоретического сценария и последующей проверки на подходящем Linux-хосте.
