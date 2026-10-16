#!/usr/bin/env python3
"""Render exact terminal-cell captures; half blocks are drawn as two rectangles.
No high-resolution artwork is substituted for the terminal image.
"""
from PIL import Image, ImageDraw, ImageFont
from pathlib import Path
import json, subprocess, argparse
ROOT=Path(__file__).resolve().parents[1]
FONT='/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf'

def rgb(n):return ((n>>16)&255,(n>>8)&255,n&255)
def picture(f,cell=8):
    w,h=f['W'],f['H'];im=Image.new('RGB',(w*cell,h*cell*2));d=ImageDraw.Draw(im)
    font=ImageFont.truetype(FONT,cell*2-3)
    for y in range(h):
        for x in range(w):
            c=f['Cells'][y*w+x];xx,yy=x*cell,y*cell*2
            d.rectangle((xx,yy,xx+cell-1,yy+cell*2-1),fill=rgb(c['BG']))
            if c['Ch']=='▀':d.rectangle((xx,yy,xx+cell-1,yy+cell-1),fill=rgb(c['FG']))
            elif c['Ch']!=' ':d.text((xx,yy-1),c['Ch'],font=font,fill=rgb(c['FG']))
    return im

def frame(scene,size,t=3,**kwargs):
    args=[str(ROOT/'bin/sea-battle'),'--scene',scene,'--size',size,'--at',str(t),'--dump','json']
    for k,v in kwargs.items():args.extend(['--'+k,str(v)])
    return json.loads(subprocess.check_output(args))

def main():
    ap=argparse.ArgumentParser();ap.add_argument('--quick',action='store_true');ap.add_argument('--motion',default='launch,spit,miss,hit,retaliation,sinking,loss,depth,human-station,alien-station,exchange,counterexchange,air-wing,brood-wing,overkill,digestive-event');args=ap.parse_args()
    out=ROOT/'captures';out.mkdir(exist_ok=True)
    scenes=[line.split()[1] for line in subprocess.check_output([str(ROOT/'bin/sea-battle'),'--list'],text=True).splitlines()]
    for size in ['hd','sd']:
        thumbs=[]
        for scene in scenes:
            im=picture(frame(scene,size));im.save(out/f'{scene}-{size}.png')
            thumb=im.resize((640,384),Image.Resampling.NEAREST);thumbs.append(thumb)
        for group in range(0,len(thumbs),8):
            chosen=thumbs[group:group+8];sheet=Image.new('RGB',(1280,384*((len(chosen)+1)//2)))
            for i,im in enumerate(chosen):sheet.paste(im,((i%2)*640,(i//2)*384))
            sheet.save(out/f'contact-{size}-{group//8}.png')
    if not args.quick:
        for scene in args.motion.split(','):
            for size in ['sd','hd']:
                times=[0,1.3,1.6,2.2,3.5,5,7,9]
                if scene in ['exchange','counterexchange']:times=[0,2,4,5.5,7,8.5,11,14]
                if scene in ['overkill','digestive-event']:times=[0,2.5,3.2,4,5,7,10,13]
                sheet=Image.new('RGB',(1280,384*4))
                for i,t in enumerate(times):
                    im=picture(frame(scene,size,t)).resize((640,384),Image.Resampling.NEAREST)
                    sheet.paste(im,((i%2)*640,(i//2)*384))
                sheet.save(out/f'motion-{scene}-{size}.png')
            duration=17 if scene in ['exchange','counterexchange'] else 14 if scene in ['overkill','digestive-event'] else 10
            movie=[picture(frame(scene,'sd',i/8)) for i in range(duration*8)]
            movie[0].save(out/f'{scene}-sd.gif',save_all=True,append_images=movie[1:],duration=125,loop=0,optimize=False)
    print(f'Captured {len(scenes)*2} native scene images and motion evidence in {out}')
if __name__=='__main__':main()
