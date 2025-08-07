#!/bin/bash
systemctl daemon-reload
systemctl enable cacophony-config-sync.service
systemctl restart cacophony-config-sync.service
