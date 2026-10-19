#!/usr/bin/env python3
"""Single-player flows in real SD/HD terminals; no network peer required."""
import json, tempfile
from pathlib import Path
from play_pty_review import Terminal
from review import ROOT

def main():
    (ROOT/'captures').mkdir(exist_ok=True)
    checks=[]
    def check(ok,label):
        assert ok,label
        checks.append(label)
    with tempfile.TemporaryDirectory(prefix='sb2-solo-') as temp:
        for faction,label in [('american','A / AMERICAN NAVY'),('alien','B / THE BROOD')]:
            for hd in [False,True]:
                t=Terminal([],Path(temp))
                try:
                    t.read(.6)
                    if hd:t.resize(160,48)
                    check('Single player' in t.read(),f'{faction} {hd}: menu entry')
                    t.key('s')
                    check('CHOOSE COMMAND' in t.read(),'Faction selection')
                    difficulty='hard' if hd else 'easy'
                    t.key('3' if hd else '1')
                    t.key('a' if faction=='american' else 'b')
                    check(label in t.read(.5) and 'SOLO '+difficulty.upper() in t.read(),'Correct faction and difficulty deployment')
                    t.key('r');t.key(' ')
                    check('YOUR TURN' in t.read(),'Human opens first match')
                    t.key('\r')
                    t.read(.8)
                    t.key(' ')
                    check('ENEMY TAKING AIM' not in t.read(.1),'Solo transition replaces the aiming modal')
                    t.read(2);t.key(' ')
                    text=t.read()
                    check('SHOT 003' in text and 'YOUR TURN' in text,'Bot responds exactly once and returns turn')
                    check('I invitation' not in text,'Solo controls omit invitation')
                    t.capture(f'solo-{faction}-{"hd" if hd else "sd"}.png')
                    t.key('\x1b');t.read(.2)
                    check('LEAVE COMMAND?' in t.read(),'Leaving unfinished game confirms')
                    t.key('n')
                    check('BATTLE' in t.read(),'Cancel leave preserves battle')
                    t.key('\x1b');t.read(.2);t.key('y')
                    check('Single player' in t.read(),'Return to command menu')
                    t.stop();checks.append('Terminal attributes restored')
                finally:t.close()
        for faction,difficulty in [('american','easy'),('american','medium'),('american','hard'),('alien','easy'),('alien','medium'),('alien','hard')]:
            t=Terminal(['--solo',faction,'--difficulty',difficulty],Path(temp))
            try:
                check('SOLO '+difficulty.upper() in t.read(.7),f'CLI starts {faction} {difficulty}')
                t.stop()
            finally:t.close()
    (ROOT/'captures/solo-pty-review.json').write_text(json.dumps({'checks':checks},indent=2)+'\n')
    print('\n'.join(checks))
if __name__=='__main__':main()
