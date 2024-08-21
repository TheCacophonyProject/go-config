#!/bin/bash
/usr/bin/cacophony-config-import
systemctl daemon-reload
systemctl enable cacophony-config-sync.service
systemctl restart cacophony-config-sync.service
