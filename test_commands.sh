./osh
echo nyan
echo nyan | cat | cat | cat
echo nyan &
./bin/sleep_echo
./bin/sleep_echo &
./bin/echo123
./bin/echo123 | cat | cat | cat
cd ..
echo nyan > neko
cat < neko < neko > neko1 > neko2
exit

a=1 ./osh 2 3 4
echo $#, $*
echo $0
echo $1
echo ~
echo <()
echo $a
