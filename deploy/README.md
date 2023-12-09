The file pik-tg-bot.service should go into Ubuntu directory /lib/systemd/system/


To start service:
sudo service pik-tg-bot start

To monitor service:
sudo journalctl -xefu pik-tg-bot
