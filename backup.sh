#!/bin/bash

echo "Début de la sauvegarde à $(date)"

#variables
SOURCE="/home/mathieul/ImportantASauvegarder"
USER="mathieu"
HOST="172.20.10.2"
DESTINATION="~/backup/"

#commande rsync
rsync -avz --delete -e "ssh -i ~/SiInfra/key.txt" $SOURCE $USER@$HOST:$DESTINATION

echo "Sauvegarde terminée à $(date)"
