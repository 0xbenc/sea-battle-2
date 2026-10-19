#!/usr/bin/env python3
"""Mouse placement, edit/undo, and launch in real SD/HD terminals."""
import os, tempfile, termios
from pathlib import Path
from play_pty_review import Terminal
from review import ROOT

def main():
    (ROOT/'captures').mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='sb2-placement-') as temp:
        # Keep this review isolated from the player's saved preferences.
        previous=os.environ.get('XDG_CONFIG_HOME')
        os.environ['XDG_CONFIG_HOME']=temp
        try:
            for hd in [False,True]:
                for team in ['american','alien']:
                    t=Terminal(['--solo',team,'--difficulty','easy'],Path(temp))
                    try:
                        t.read(1)
                        if hd:t.resize(160,48)
                        text=t.read(.3)
                        assert 'YOUR WATERS' in text and 'HOSTILE WATERS' not in text
                        x,y,cw,ch,side,actions=(12,14,8,6,104,31) if hd else (5,8,4,3,49,16)
                        def click(px,row,button=0):
                            prefix=f'\x1b[<{button};{px+1};{row+1}'
                            t.key(prefix+'M');t.key(prefix+'m')
                            return t.read(.15)
                        def cell(col,row):
                            return click(x+col*cw+cw//2,(y+row*ch+ch//2)//2)
                        for i,col in enumerate([0,2,4,6,8]):
                            text=cell(col,0)
                            assert f'FLEET  {i+1} / 5' in text,(hd,team,'place',i)
                        assert 'Fleet ready.' in text
                        # Undo the last placement using the visible button, then replace it.
                        text=click(side+(22 if hd else 14),actions+2)
                        assert 'FLEET  4 / 5' in text,'undo'
                        assert 'Fleet ready.' in cell(8,0),'replace after undo'
                        # Pick up the carrier by its middle and move its bow to J1.
                        cell(8,2)
                        text=cell(9,0)
                        assert 'FLEET  5 / 5' in text and 'Fleet ready.' in text,'edit ship'
                        t.capture(f'placement-ready-{team}-{"hd" if hd else "sd"}.png')
                        t.key('?');assert 'PLACEMENT CONTROLS' in t.read(.2),'help'
                        t.key('?');t.read(.2)
                        text=click(side+2,actions+4)
                        assert 'YOUR TURN' in text and 'HOSTILE WATERS' in text,'launch into battle'
                        t.key('\x03');t.read(1);t.proc.wait(timeout=5)
                        assert termios.tcgetattr(t.slave)==t.original
                    finally:
                        if t.proc.poll() is None:t.proc.kill();t.proc.wait()
                        t.close()
        finally:
            if previous is None:os.environ.pop('XDG_CONFIG_HOME',None)
            else:os.environ['XDG_CONFIG_HOME']=previous
    print('Both factions at SD/HD: manual mouse placement, undo, move, help, launch, and terminal restoration passed.')

if __name__=='__main__':main()
