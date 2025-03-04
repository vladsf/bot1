**Steps to Use:**

1. Move the `bot1.service` file to the `/etc/systemd/system/` directory:
    * ```bash
        sudo mv bot1.service /etc/systemd/system/
        ```

2. Tell systemd to reload its configuration files:
    * ```bash
        sudo systemctl daemon-reload
        ```

3. Enable the service to start automatically on boot:
    * ```bash
        sudo systemctl enable bot1.service
        ```

4. Start the service immediately:
    * ```bash
        sudo systemctl start bot1.service
        ```

5. Verify that the service is running correctly:
    * ```bash
        sudo systemctl status bot1.service
        ```

6. View the logs:
    * ```bash
        sudo journalctl -u bot1.service -f
        ```
    This command is very useful for viewing the logs of your bot. 
    The `-f` flag makes it continuously follow the log output, so you can see updates in real-time.
