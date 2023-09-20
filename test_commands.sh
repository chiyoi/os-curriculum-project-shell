./osh
echo nyan
echo nyan | cat | cat | cat
./echo123
./echo123 | cat | cat | cat
echo nyan &
./sleep_echo
./sleep_echo &
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
