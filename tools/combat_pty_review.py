#!/usr/bin/env python3
"""Real terminal mouse targeting and chart inspection, both factions and sizes."""
import os, tempfile
from pathlib import Path
from play_pty_review import Terminal

def click(t, x, y):
    t.key(f'\x1b[<0;{x+1};{y+1}M')
    return t.key(f'\x1b[<0;{x+1};{y+1}m')

def main():
    with tempfile.TemporaryDirectory(prefix='sb2-combat-') as temp:
        os.environ['XDG_CONFIG_HOME'] = temp
        for faction in ['american', 'alien']:
            for hd in [False, True]:
                t = Terminal(['--solo', faction], Path(temp))
                try:
                    t.read(.5)
                    if hd: t.resize(160, 48)
                    t.key('r'); t.key(' ')
                    assert 'YOUR TURN' in t.read()
                    # Expand your chart, then return via the small enemy chart.
                    text = click(t, 112 if hd else 55, 10 if hd else 6)
                    assert 'TAB Targets' in text and 'SHOT 001' in text
                    text = click(t, 112 if hd else 55, 10 if hd else 6)
                    assert 'ENTER Fire A' in text and 'SHOT 001' in text
                    # A fleet name focuses the actual ship; Enter only returns.
                    text = click(t, 105 if hd else 50, 27 if hd else 16)
                    assert 'intact' in text
                    assert 'SHOT 001' in t.key('\r')
                    t.key('?')
                    assert 'FIRE CONTROL' in t.read()
                    assert 'SHOT 001' in click(t, 15, 8)
                    # Aim B4, then fire exactly once on mouse press/release.
                    x, y = (24, 17) if hd else (11, 9)
                    text = t.key(f'\x1b[<35;{x+1};{y+1}M')
                    assert 'ENTER Fire B4' in text
                    click(t, x, y)
                    t.read(.5); t.key(' ')
                    assert 'ENEMY TAKING AIM' not in t.read(.1)
                    t.read(2); t.key(' ')
                    text = t.read()
                    assert 'SHOT 003' in text and 'YOUR TURN' in text
                    assert 'B4 /' in text
                    # Repeated clicking the resolved cell cannot fire again.
                    text = click(t, x, y)
                    assert 'SHOT 003' in text and 'YOUR TURN' in text
                    t.capture(f'combat-mouse-{faction}-{"hd" if hd else "sd"}.png')
                    t.stop()
                    print(f'{faction} {"HD" if hd else "SD"}: inspect, help, aim, fire, bot response, repeat gate passed')
                finally:
                    t.close()

if __name__ == '__main__': main()
