if [ -f ./NPC ]; then
  rm ./NPC
fi

go build -v -o NPC

if [ -f ./NPC ]; then
  source ./env.sh
  ./NPC
fi
