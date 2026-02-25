#!/usr/bin/env bash

download() {
  ./bin/spoticsv -csv "$1" -yes -format HI_RES -lyrics -output /mnt/prodigy/mojo/audio/music/spotiflac
}

#files=$(fd 'csv$' /mnt/prodigy/general/backups/exportify)
files=$(cat playlistlist)


for file in $files; 
do
	download $file
done
