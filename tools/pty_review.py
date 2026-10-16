#!/usr/bin/env python3
"""Exercise the executable in a real PTY and compare its cells to deterministic renders."""
import codecs, fcntl, json, os, pty, select, signal, struct, subprocess, termios, time
from pathlib import Path
import pyte
from review import picture, frame, ROOT

class Screen(pyte.Screen):
    last_graphic=' '
    def draw(self,data):
        super().draw(data)
        if data:self.last_graphic=data[-1]
    def repeat_character(self,count=1):self.draw(self.last_graphic*max(1,count))
    def backward_tab(self,count=1):
        for _ in range(max(1,count)):self.cursor.x=max([0]+[s for s in self.tabstops if s<self.cursor.x])
    def forward_tab(self,count=1):
        for _ in range(max(1,count)):self.tab()
    def reset_tabs(self,mode=0,private=False):
        if private and mode==5:self.tabstops=set(range(8,self.columns,8))
    def delete_characters(self,count=1):
        count=count or 1;super().delete_characters(count)
        for x in range(max(self.cursor.x,self.columns-count),self.columns):self.buffer[self.cursor.y][x]=self.cursor.attrs._replace(data=' ')
class Stream(pyte.Stream):
    csi=dict(pyte.Stream.csi,b='repeat_character',Z='backward_tab',I='forward_tab',W='reset_tabs')
    events=pyte.Stream.events|{'repeat_character','backward_tab','forward_tab','reset_tabs'}

def main():
    out=ROOT/'captures';out.mkdir(exist_ok=True)
    master,slave=pty.openpty();fcntl.ioctl(slave,termios.TIOCSWINSZ,struct.pack('HHHH',48,160,0,0));original=termios.tcgetattr(slave)
    proc=subprocess.Popen([str(ROOT/'bin/sea-battle'),'--scene','human-station','--paused','--at','3','--tour=false'],stdin=slave,stdout=slave,stderr=slave,env=dict(os.environ,TERM='xterm-256color',COLORTERM='truecolor'))
    screen=Screen(160,48);stream=Stream(screen);decoder=codecs.getincrementaldecoder('utf8')();transcript=bytearray();checks=[]
    def read(seconds=.25):
        end=time.monotonic()+seconds
        while time.monotonic()<end:
            if select.select([master],[],[],max(0,end-time.monotonic()))[0]:
                try:data=os.read(master,1<<20)
                except OSError:break
                transcript.extend(data);stream.feed(decoder.decode(data))
        return '\n'.join(screen.display)
    def key(s):os.write(master,s.encode());return read()
    def resize(w,h):
        fcntl.ioctl(slave,termios.TIOCSWINSZ,struct.pack('HHHH',h,w,0,0));screen.resize(h,w);proc.send_signal(signal.SIGWINCH);return read()
    def check(cond,label):
        if not cond:raise AssertionError(label)
        checks.append(label)
    def snapshot(name):
        cells=[]
        for y in range(screen.lines):
            for x in range(screen.columns):
                c=screen.buffer[y][x]
                def color(s):
                    try:return int(s,16)
                    except ValueError:return 0
                cells.append(dict(Ch=c.data,FG=color(c.fg),BG=color(c.bg)))
        f=dict(W=screen.columns,H=screen.lines,Cells=cells);picture(f).save(out/(name+'.png'));return f
    def pixels():
        return [(screen.buffer[y][x].data,screen.buffer[y][x].fg,screen.buffer[y][x].bg) for y in range(screen.lines) for x in range(screen.columns)]
    try:
        read(1)
        check('GENERAL QUARTERS' in '\n'.join(screen.display),'HD startup')
        snapshot('pty-hd')
        key('\x1bOQ');actual=snapshot('pty-sd-centered');check('SD' in '\n'.join(screen.display),'F2 prefers SD')
        resize(80,24);actual=snapshot('pty-sd')
        expected=frame('human-station','sd',3)
        # CLI dump advertises TOUR and no PAUSED; all image pixels outside chrome must match.
        mismatch=[]
        for y in range(2,22):
            for x in range(80):
                a,b=actual['Cells'][y*80+x],expected['Cells'][y*80+x]
                if a!=b:mismatch.append((x,y,a,b))
        check(not mismatch,f'PTY cells match renderer ({len(mismatch)} mismatches)')
        key('d');check('TARGET  H5' in '\n'.join(screen.display),'Aim changes target')
        key('\t');check('SCENE DECK' in '\n'.join(screen.display),'Scene deck opens')
        key('\x1b[F');key('\r');check('CINEMATIC EXCESS' in '\n'.join(screen.display),'Scene deck selects last scene')
        key('?');check('LOOKDEV CONTROLS' in '\n'.join(screen.display),'Help opens');key('\x1b')
        resize(70,20);check('Need 80 columns' in '\n'.join(screen.display),'Undersize guidance')
        resize(160,48);check('LOOKDEV / SD' in '\n'.join(screen.display),'Explicit SD preference survives resize')
        key('\x1bOQ');check('LOOKDEV / HD' in '\n'.join(screen.display),'F2 returns to HD')
        resize(80,24);check('LOOKDEV / SD' in '\n'.join(screen.display),'Automatic HD fallback')
        resize(160,48);check('LOOKDEV / HD' in '\n'.join(screen.display),'Automatic HD recovery')
        key(' ');before=pixels();read(.5);check(before!=pixels(),'Live animation advances')
        key(' ');before=pixels();read(.4);check(before==pixels(),'Pause holds exact frame')
        key('q');proc.wait(timeout=3);check(proc.returncode==0,'Clean quit');check(termios.tcgetattr(slave)==original,'Terminal attributes restored')
        (out/'pty-review.json').write_text(json.dumps({'checks':checks},indent=2)+'\n');print('\n'.join(checks))
    finally:
        if proc.poll() is None:proc.terminate();proc.wait()
        (out/'pty-transcript.bin').write_bytes(transcript);os.close(master);os.close(slave)
if __name__=='__main__':main()
