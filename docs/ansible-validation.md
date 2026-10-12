# Проверка Ansible на подходящем Linux-хосте

Этот документ описывает практическую проверку, которую нужно выполнить, если станет доступна Linux VM с SSH. Наличие файлов Ansible само по себе не означает, что playbook был выполнен.

## Подготовка контроллера

Из корня репозитория на Linux/WSL-контроллере:

```bash
sudo apt update
sudo apt install -y ansible openssh-client
ansible-galaxy collection install community.docker
```

Затем отредактируйте `ansible/inventory.ini`, указав IP-адрес гостевой Ubuntu и пользователя, которому доступен SSH. Настройте ключ SSH и проверьте подключение:

```bash
ansible -i ansible/inventory.ini weather_server -m ansible.builtin.ping
```

Ожидается `ping: pong` и `unreachable=0`.

## Первый запуск

```bash
ansible-playbook -i ansible/inventory.ini ansible/deploy.yml --ask-become-pass
```

Команда запросит пароль sudo для пользователя на VM. Успешный итог должен содержать `failed=0` и `unreachable=0`.

## Повторный запуск для проверки идемпотентности

Не меняя файлы и конфигурацию, повторите ту же команду:

```bash
ansible-playbook -i ansible/inventory.ini ansible/deploy.yml --ask-become-pass
```

При неизменном состоянии целевой системы ожидается успешный повторный запуск и, как правило, `changed=0`. Если `changed` остаётся ненулевым, исследуйте, какая задача сообщила об изменении; не заявляйте идемпотентность без проверки.

## Проверка приложения на VM

```bash
sudo docker compose -f /opt/weatherwise/compose.yaml ps
```

Проверьте опубликованный порт фронтенда в `/opt/weatherwise/compose.yaml`, затем откройте `http://<ip-адрес-vm>:<порт>` в браузере контроллера/ноутбука. В браузере через Developer Tools → Network следует проверить успешный запрос к backend API.

## Предпосылки

Playbook предполагает, что корень проекта содержит `compose.yaml`, `frontend/` и `backend/`; дополнительные файлы конфигурации, например `.env`, при наличии нужно явно доставить отдельной задачей и не коммитить секреты. Для управления Compose на контроллере требуется коллекция `community.docker`; на целевой VM устанавливаются пакеты `docker.io` и `docker-compose-v2`.
