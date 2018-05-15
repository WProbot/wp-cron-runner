# WordPress cron runner
WordPress cron runner example for the VIP multisite environment.

The runner iterates over active WordPress sites in the multisite installation and executes scheduled cron events using WP CLI. The runner can span multiple threads to run cron on several websites at once.
