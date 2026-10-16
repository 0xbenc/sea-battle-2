#!/usr/bin/env python3
"""Two real terminals: host/paste invitation, ready, fire, reconnect, resize."""
import codecs, fcntl, json, os, pty, select, signal, struct, subprocess, tempfile, termios, time
from pathlib import Path
from pty_review import Screen, Stream
from review import picture, ROOT

class Terminal:
    def __init__(self,args,cwd):
        self.master,self.slave=pty.openpty()
        fcntl.ioctl(self.slave,termios.TIOCSWINSZ,struct.pack('HHHH',24,80,0,0))
        self.original=termios.tcgetattr(self.slave)
        self.proc=subprocess.Popen([str(ROOT/'bin/sea-battle'),*args],cwd=cwd,stdin=self.slave,stdout=self.slave,stderr=self.slave,env=dict(os.environ,TERM='xterm-256color',COLORTERM='truecolor'))
        self.screen=Screen(80,24);self.stream=Stream(self.screen);self.decoder=codecs.getincrementaldecoder('utf8')()
    def read(self,seconds=.2):
        end=time.monotonic()+seconds
        while time.monotonic()<end:
            if select.select([self.master],[],[],max(0,end-time.monotonic()))[0]:
                try:data=os.read(self.master,1<<20)
                except OSError:break
                self.stream.feed(self.decoder.decode(data))
        return '\n'.join(self.screen.display)
    def key(self,key):
        os.write(self.master,key.encode());return self.read()
    def resize(self,w,h):
        fcntl.ioctl(self.slave,termios.TIOCSWINSZ,struct.pack('HHHH',h,w,0,0))
        self.screen.resize(h,w);self.proc.send_signal(signal.SIGWINCH);return self.read()
    def capture(self,name):
        cells=[]
        for y in range(self.screen.lines):
            for x in range(self.screen.columns):
                c=self.screen.buffer[y][x]
                def color(s):
                    try:return int(s,16)
                    except ValueError:return 0
                cells.append(dict(Ch=c.data,FG=color(c.fg),BG=color(c.bg)))
        picture(dict(W=self.screen.columns,H=self.screen.lines,Cells=cells)).save(ROOT/'captures'/name)
    def stop(self):
        if self.proc.poll() is None:
            self.key('\x03');self.proc.wait(timeout=5)
        assert self.proc.returncode==0
        assert termios.tcgetattr(self.slave)==self.original
    def close(self):
        if self.proc.poll() is None:self.proc.terminate();self.proc.wait(timeout=5)
        os.close(self.master);os.close(self.slave)

def main():
    (ROOT/'captures').mkdir(exist_ok=True)
    checks=[];terminals=[]
    def check(ok,label):
        assert ok,label
        checks.append(label)
    with tempfile.TemporaryDirectory(prefix='sb2-pty-') as temp:
        temp=Path(temp)
        try:
            a=Terminal(['--host','--no-map','--port','0'],temp);terminals.append(a)
            check('HOST INVITATION' in a.read(1),'Host invitation screen')
            a.key('s');invite=(temp/'invitation.txt').read_text()
            check(invite.startswith('sb2:'),'Private invitation file')
            check((temp/'invitation.txt').stat().st_mode&0o777==0o600,'Invitation file permissions')
            b=Terminal([],temp);terminals.append(b)
            check('Host a battle' in b.read(1),'Default Host/Join menu')
            b.key('j');b.key('\x1b[200~'+invite+'\x1b[201~')
            b.key('\r')
            check('B / THE BROOD' in b.read(.8),'Pasted invitation joins alien seat')
            check('FRIENDLY WATERS' in a.read(.3),'Host automatically deploys when guest joins')
            a.key('r');a.key(' ')
            check('FLEET LOCKED' in a.read(.3),'Host fleet locks')
            b.key('r');b.key(' ')
            sa,sb=a.read(.5),b.read(.5)
            check('BATTLE' in sa and 'BATTLE' in sb,'Both fleets start live battle')
            first=a if 'YOUR TURN' in sa else b
            other=b if first is a else a
            first.key('\r');a.read(.3);b.read(.3)
            check('A1' in a.read() and 'A1' in b.read(),'Same real shot coordinate in both cinematics')
            a.key(' ');b.key(' ')
            sa,sb=a.read(),b.read()
            check('LAST: A1' in sa and 'LAST: A1' in sb,'Both charts confirm the same shot')
            check('YOUR TURN' in other.read(),'Turn passes after one shot')
            other.key('d');other.key('\r');a.read(.2);b.read(.2);a.key(' ');b.key(' ')
            check('LAST: B1' in a.read() and 'LAST: B1' in b.read(),'Second turn resolves through server')
            a.capture('live-pty-human-sd.png');b.capture('live-pty-alien-sd.png')
            check('Need 80 x 24' in a.resize(70,20),'Undersize gates play')
            check('FRIENDLY WATERS' in a.resize(160,48),'HD recovers after resize')
            a.capture('live-pty-human-hd.png')
            b.stop()
            check('LINK LOST' in a.read(.5),'Disconnect pauses battle')
            b=Terminal(['--join-file',str(temp/'invitation.txt')],temp);terminals.append(b)
            check('BATTLE' in b.read(1),'Same invitation reconnects to existing match')
            check('LAST: B1' in b.read(),'Reconnect restores shot history')
            a.read(.3);a.key('q')
            check('Closing the host ends' in a.read(),'Host quit confirmation explains session lifetime')
            a.key('n');a.stop();b.stop()
            checks.append('Both terminals restore terminal attributes')
        finally:
            for t in terminals:t.close()
    (ROOT/'captures/live-pty-review.json').write_text(json.dumps(dict(checks=checks),indent=2)+'\n')
    print('\n'.join(checks))
if __name__=='__main__':main()
