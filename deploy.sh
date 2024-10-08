#!/bin/bash

# Start the SSH agent
eval "$(ssh-agent -s)"
ssh_key_location="$HOME/.ssh/cacophony-pi"
ssh-add "$ssh_key_location"

# Discover Raspberry Pi services on the network
echo "Discovering Raspberry Pis with service _cacophonator-management._tcp..."
readarray -t services < <(avahi-browse -t -r _cacophonator-management._tcp | grep 'hostname' | awk '{print $3}' | sed 's/\[//' | sed 's/\]//' | sed 's/\.$/.local/')

if [ ${#services[@]} -eq 0 ]; then
	echo "No Raspberry Pi services found on the network."
	exit 1
fi

# Display the discovered services
echo "Found Raspberry Pi services:"
for i in "${!services[@]}"; do
	echo "$((i + 1))) ${services[i]}"
done

# Let the user select a service
read -p "Select a Raspberry Pi to deploy to (1-${#services[@]}): " selection
pi_address=${services[$((selection - 1))]}

if [ -z "$pi_address" ]; then
	echo "Invalid selection."
	exit 1
fi

echo "Selected Raspberry Pi at: $pi_address"

while true; do
	# Deployment
	echo "Deploying to Raspberry Pi..."
	make

	# Create the service file if it doesn't exist
	ssh -i "$ssh_key_location" "pi@$pi_address" "if [ ! -f /etc/systemd/system/cacophony-config-sync.service ]; then
        sudo tee /etc/systemd/system/cacophony-config-sync.service > /dev/null << 'EOF'
[Unit]
Description=Cacophony Config Sync Service
After=network.target

[Service]
ExecStart=/usr/bin/cacophony-config-sync
Restart=always
User=pi
Group=pi

[Install]
WantedBy=multi-user.target
EOF
        sudo systemctl daemon-reload
        sudo systemctl enable cacophony-config-sync.service
    fi"

	# Stop the service
	ssh_stop_command=("ssh" "-i" "$ssh_key_location" "pi@$pi_address" "sudo systemctl stop cacophony-config-sync.service")
	echo "Executing: ${ssh_stop_command[*]}"
	"${ssh_stop_command[@]}"
	if [ $? -ne 0 ]; then
		echo "Error: SSH stop command failed"
		break
	fi

	# Copy the file to a temporary location
	scp_command=("scp" "-i" "$ssh_key_location" "./cacophony-config-sync" "pi@$pi_address:/home/pi")
	echo "Executing: ${scp_command[*]}"
	"${scp_command[@]}"
	if [ $? -ne 0 ]; then
		echo "Error: SCP failed"
		break
	fi

	# Move the file to /usr/bin with sudo
	ssh_move_command=("ssh" "-i" "$ssh_key_location" "pi@$pi_address" "sudo mv /home/pi/cacophony-config-sync /usr/bin/")
	echo "Executing: ${ssh_move_command[*]}"
	"${ssh_move_command[@]}"
	if [ $? -ne 0 ]; then
		echo "Error: SSH move command failed"
		break
	fi

	# Restart the service
	ssh_command=("ssh" "-i" "$ssh_key_location" "pi@$pi_address" "sudo systemctl restart cacophony-config-sync.service")
	echo "Executing: ${ssh_command[*]}"
	"${ssh_command[@]}"
	if [ $? -ne 0 ]; then
		echo "Error: SSH command failed"
		break
	fi

	# Stream logs from the service
	log_command=("ssh" "-i" "$ssh_key_location" "pi@$pi_address" "sudo journalctl -fu cacophony-config-sync.service ")
	echo "Streaming logs from cacophony-config-sync.service... (press Ctrl+C to stop)"
	"${log_command[@]}"

	echo "Deployment completed. Press any key to deploy again or Ctrl+C to exit."
	read -n 1 -s
	if [ $? -ne 0 ]; then
		break
	fi
	echo # new line for readability
done

# Kill the SSH agent when done
eval "$(ssh-agent -k)"
