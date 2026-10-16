#!/usr/bin/env python3
"""Author native pixel silhouettes as Dotmason v1. No image scaling at runtime.
Each size is rasterized on its own grid, with explicit detail thresholds.
"""
from PIL import Image, ImageDraw
from pathlib import Path
import json, math

ROOT = Path(__file__).resolve().parents[1]
NAMES = ['Cyclone','Virginia','Freedom','Zumwalt','Gerald R. Ford',
         'Drone Skiff','Deep Lurker','Littoral Ravager','Siege Carapace','Hive Carrier']
LENGTHS = [2,3,3,4,5]*2
# Palette identities stay distinct even where their defaults might later match.
PALETTE = ['#101F2B','#284453','#476775','#75949D','#ACC5C3','#E8E5C9',
           '#D9AC65','#151A24','#2C2537','#4C354A','#755266','#A87B81',
           '#DBC7A3','#33494C','#587974','#8FAD99','#66E9BF','#C3FFE0',
           '#F06D41','#FFC87B','#FFF1C9','#1C343F','#F2D583','#ACC5C3']

class Painter:
    def __init__(self,w,h):
        self.im=Image.new('I',(w,h));self.d=ImageDraw.Draw(self.im);self.w=w;self.h=h
    def pt(self,x,y):return (round(x*(self.w-1)),round(y*(self.h-1)))
    def poly(self,pts,c):self.d.polygon([self.pt(*p) for p in pts],fill=c)
    def box(self,x,y,w,h,c):self.d.rectangle([self.pt(x,y),self.pt(x+w,y+h)],fill=c)
    def line(self,pts,c,width=1):self.d.line([self.pt(*p) for p in pts],fill=c,width=width)
    def oval(self,x,y,w,h,c):self.d.ellipse([self.pt(x,y),self.pt(x+w,y+h)],fill=c)

def side(p,i):
    alien=i>=5;k=i%5
    if not alien:
        # Common keel, individually identifiable superstructures above it.
        p.poly([(.04,.68),(.96,.65),(.89,.87),(.13,.87),(.06,.79)],2)
        p.poly([(.04,.65),(.96,.65),(.91,.77),(.09,.77)],3)
        p.line([(.09,.78),(.9,.78)],4)
        p.line([(.15,.88),(.87,.88)],1)
        if k==0:
            p.box(.34,.43,.26,.22,3);p.box(.4,.32,.16,.12,4)
            p.box(.45,.25,.05,.08,3);p.line([(.48,.25),(.48,.08),(.6,.08)],5)
            p.box(.64,.6,.09,.05,5);p.line([(.68,.6),(.77,.53)],5)
            p.box(.1,.61,.13,.04,4)
        elif k==1:
            p.oval(.03,.57,.93,.26,2);p.oval(.08,.57,.85,.11,3)
            p.poly([(.36,.59),(.39,.34),(.5,.32),(.55,.6)],3)
            p.line([(.42,.34),(.42,.17),(.46,.17)],5)
            p.line([(.5,.34),(.5,.22)],4)
            p.poly([(.15,.6),(.12,.43),(.09,.61)],4)
            p.poly([(.87,.63),(.92,.48),(.92,.67)],4)
        elif k==2:
            p.poly([(.25,.65),(.32,.32),(.53,.32),(.68,.65)],4)
            p.poly([(.28,.65),(.34,.41),(.53,.41),(.63,.65)],3)
            p.line([(.43,.32),(.43,.11),(.55,.11)],5)
            p.box(.15,.59,.12,.05,5);p.line([(.19,.58),(.26,.49)],5)
            p.box(.7,.61,.16,.025,5)
        elif k==3:
            p.poly([(.28,.65),(.41,.24),(.65,.24),(.77,.65)],3)
            p.poly([(.41,.24),(.65,.24),(.7,.41),(.36,.41)],4)
            p.line([(.43,.25),(.61,.25)],5)
            p.box(.14,.59,.11,.06,4);p.line([(.2,.6),(.28,.56)],5)
            p.box(.78,.61,.09,.04,4)
        else:
            p.poly([(.02,.58),(.97,.58),(.94,.68),(.04,.68)],4)
            p.line([(.03,.58),(.96,.58)],6)
            p.box(.62,.34,.14,.24,3);p.box(.59,.34,.2,.04,5)
            p.box(.65,.24,.1,.1,4);p.line([(.69,.24),(.69,.07),(.76,.07)],5)
            for x in [.13,.27,.41,.85]:
                p.line([(x-.025,.54),(x+.04,.54)],5)
                p.poly([(x,.49),(x+.025,.54),(x-.025,.54)],6)
        # Bridge glass, radar lights and deck fittings at native resolution.
        if k!=1:
            if k==4:x,y=.63,.38
            elif k==3:x,y=.42,.33
            elif k==2:x,y=.35,.36
            else:x,y=.42,.37
            p.line([(x,y),(x+.09,y)],1)
            if p.w>60:
                for q in range(4):p.box(x+q*.025,y,.009,.02,5)
        if k!=1:
            for x in [.2,.31,.44,.57,.73]:
                p.box(x,.71,.008,.01,1)
            if p.w>60:
                p.line([(.1,.63),(.29,.63)],5)
                for x in [.11,.17,.23]:p.line([(x,.6),(x,.65)],4)
                p.line([(.3,.82),(.8,.82)],2)
                p.line([(.78,.65),(.9,.65)],5)
        else:
            p.line([(.18,.73),(.84,.73)],1)
            if p.w>60:
                for x in [.27,.33,.63,.69]:p.line([(x,.62),(x+.025,.65)],1)
        p.box(.14,.8,.04,.02,6)
    else:
        # Chitin keel, muscle underbelly, articulated armored ribs, sparse implants.
        p.poly([(.03,.67),(.14,.52),(.28,.49),(.42,.56),(.62,.5),(.85,.54),(.98,.66),(.87,.86),(.21,.87)],9)
        p.oval(.13,.61,.76,.27,10)
        p.line([(.18,.82),(.37,.87),(.66,.85),(.88,.75)],11)
        for j in range(5+k):
            x=.16+j*(.64/(5+k))
            p.poly([(x,.66),(x+.025,.43- .05*math.sin(j)),(x+.08,.53),(x+.12,.74),(x+.07,.8)],11)
            p.line([(x+.025,.45),(x+.07,.57),(x+.1,.72)],12)
        if k==0:
            p.poly([(.65,.56),(.83,.37),(.94,.43),(.8,.62)],12)
            p.line([(.2,.7),(.09,.89),(.02,.91)],13)
        elif k==1:
            p.oval(.19,.58,.64,.19,14)
            p.poly([(.4,.57),(.45,.3),(.51,.57)],12)
            p.line([(.47,.36),(.49,.19),(.56,.22)],17)
        elif k==2:
            p.poly([(.52,.55),(.58,.28),(.81,.22),(.84,.3),(.67,.4),(.66,.62)],11)
            p.oval(.77,.21,.09,.12,14);p.oval(.79,.23,.05,.06,17)
        elif k==3:
            p.poly([(.28,.54),(.3,.27),(.45,.12),(.66,.19),(.72,.46)],11)
            p.line([(.32,.27),(.47,.18),(.64,.23),(.67,.4)],13)
            p.box(.44,.24,.12,.08,14);p.box(.46,.25,.08,.035,17)
            p.poly([(.73,.59),(.86,.35),(.98,.32),(.9,.52)],12)
        else:
            p.oval(.22,.26,.55,.43,10)
            for j in range(4):
                x=.27+j*.12
                p.oval(x,.35,.1,.2,11);p.line([(x+.02,.37),(x+.04,.28)],13)
                p.oval(x+.025,.4,.04,.065,17)
            p.poly([(.1,.6),(.06,.35),(.15,.48),(.23,.59)],13)
            p.poly([(.78,.59),(.85,.3),(.9,.47),(.96,.52)],13)
        # Two technology grafts, not an all-metal alien skin.
        p.poly([(.25,.67),(.31,.59),(.4,.62),(.41,.76),(.3,.79)],14)
        p.line([(.28,.68),(.34,.65),(.37,.73)],17)
        p.box(.65,.69,.1,.06,14);p.line([(.67,.7),(.72,.7)],17)
        p.line([(.86,.64),(.94,.67),(.87,.7)],18)
        if p.w>60:
            for j in range(5+k):
                x=.16+j*(.64/(5+k))
                p.line([(x+.045,.58),(x+.067,.63),(x+.073,.7)],10)
                p.box(x+.06,.59,.006,.012,13)
            for j in range(9):
                x=.2+j*.065;p.line([(x,.83),(x+.025,.77)],12)

def top(p,i):
    k=i%5;alien=i>=5
    if alien and k==0:
        p.poly([(.5,.03),(.7,.24),(.75,.69),(.95,.92),(.62,.87),(.5,.98),(.38,.87),(.05,.92),(.25,.69),(.3,.24)],10)
        p.oval(.3,.26,.4,.48,11);p.line([(.5,.14),(.5,.78)],13)
        p.box(.4,.4,.2,.12,14);p.box(.45,.42,.1,.04,17)
        return
    if alien and k==1:
        p.oval(.29,.03,.42,.92,10);p.oval(.34,.14,.32,.72,11)
        p.line([(.5,.08),(.5,.91)],12)
        p.poly([(.3,.36),(.04,.49),(.3,.46)],13)
        p.poly([(.7,.36),(.96,.49),(.7,.46)],13)
        p.oval(.4,.38,.2,.16,14);p.line([(.48,.39),(.48,.49)],17)
        p.line([(.3,.85),(.7,.85)],14)
        return
    if not alien:
        p.poly([(.5,.015),(.8,.14),(.9,.76),(.75,.97),(.25,.97),(.1,.76),(.2,.14)],2)
        p.poly([(.5,.035),(.72,.17),(.78,.87),(.68,.93),(.32,.93),(.22,.87),(.28,.17)],4)
        p.line([(.5,.07),(.5,.94)],5)
        if k==1:
            p.oval(.24,.12,.52,.76,2);p.oval(.38,.37,.25,.22,4)
            p.line([(.12,.3),(.88,.3)],3)
        elif k==4:
            p.poly([(.2,.1),(.8,.1),(.92,.79),(.73,.94),(.12,.94)],3)
            p.line([(.33,.14),(.33,.86)],6);p.line([(.49,.18),(.49,.89)],5)
            p.box(.67,.38,.13,.23,5)
            for y in [.25,.44,.64,.79]:
                p.line([(.16,y),(.27,y)],6)
        else:
            p.box(.32,.38,.36,.32,3);p.box(.36,.42,.28,.13,5)
            p.oval(.39,.22,.22,.09,3);p.line([(.5,.23),(.5,.13)],6)
            if k==3:p.poly([(.5,.32),(.7,.51),(.65,.68),(.35,.68),(.3,.51)],3)
            p.box(.3,.8,.4,.07,2)
    else:
        p.poly([(.5,.01),(.8,.17),(.92,.56),(.83,.85),(.5,.98),(.17,.85),(.08,.56),(.2,.17)],9)
        p.oval(.2,.12,.6,.79,11)
        for j in range(6):
            y=.18+j*.11
            p.poly([(.48,y),(.24,y-.035),(.07,y+.03),(.23,y+.07),(.47,y+.1)],12)
            p.poly([(.52,y),(.76,y-.035),(.93,y+.03),(.77,y+.07),(.53,y+.1)],10)
        p.line([(.5,.17),(.5,.85)],13)
        p.oval(.33,.26,.34,.14,14);p.oval(.43,.3,.12,.045,17)
        if k==4:
            for y in [.48,.63,.78]:
                p.oval(.27,y,.18,.08,17);p.oval(.56,y,.18,.08,17)
        elif k==3:p.oval(.22,.4,.56,.3,14);p.line([(.5,.46),(.5,.12)],17)
        elif k==2:
            p.poly([(.4,.45),(.32,.16),(.35,.02),(.65,.02),(.68,.16),(.6,.45)],12)
            p.oval(.4,.03,.2,.1,17)
        else:p.line([(.62,.56),(.72,.64),(.62,.73)],17)

def front(p,i):
    alien=i>=5;k=i%5
    if alien and k==0:
        p.poly([(.12,.76),(.28,.45),(.5,.37),(.72,.45),(.88,.76),(.5,.93)],11)
        p.line([(.16,.7),(.06,.89),(.28,.84)],13);p.line([(.84,.7),(.94,.89),(.72,.84)],13)
        p.oval(.3,.58,.13,.1,17);p.oval(.57,.58,.13,.1,17)
        p.line([(.5,.4),(.5,.86)],12)
        return
    if alien and k==1:
        p.oval(.2,.5,.6,.35,10);p.oval(.31,.56,.38,.21,11)
        p.poly([(.4,.54),(.45,.2),(.55,.2),(.6,.54)],12)
        p.line([(.1,.65),(.9,.65)],14);p.line([(.4,.65),(.6,.65)],17)
        return
    if not alien:
        p.poly([(.12,.56),(.88,.56),(.72,.86),(.5,.96),(.28,.86)],2)
        p.poly([(.12,.56),(.5,.67),(.88,.56),(.72,.81),(.5,.94)],3)
        p.line([(.5,.67),(.5,.94)],5)
        if k==1:
            p.oval(.16,.49,.68,.34,2);p.box(.44,.21,.12,.35,3)
        elif k==4:
            p.box(.02,.5,.96,.05,4);p.box(.7,.23,.18,.27,3)
            p.line([(.8,.23),(.8,.06)],5)
        else:
            p.poly([(.26,.56),(.35,.27),(.65,.27),(.74,.56)],4)
            p.box(.37,.32,.26,.045,1)
            p.line([(.5,.27),(.5,.06)],5);p.line([(.36,.15),(.64,.15)],5)
            p.oval(.43,.51,.14,.09,5)
    else:
        p.oval(.12,.3,.76,.58,10)
        for j in range(5):
            x=.15+j*.145
            p.poly([(x,.61),(x+.02,.25),(x+.09,.16),(x+.14,.68)],11)
            p.line([(x+.09,.19),(x+.1,.55)],13)
        p.poly([(.12,.53),(.01,.7),(.21,.93),(.33,.89),(.15,.75)],13)
        p.poly([(.88,.53),(.99,.7),(.79,.93),(.67,.89),(.85,.75)],13)
        p.oval(.31,.51,.38,.21,9);p.line([(.36,.6),(.64,.6)],17)
        p.box(.41,.32,.18,.09,14);p.box(.46,.34,.08,.03,17)
        if k==2:
            p.oval(.33,.22,.34,.43,13);p.oval(.38,.26,.24,.3,9);p.oval(.42,.32,.16,.16,17)
        if k==4:
            p.line([(.15,.53),(.08,.17),(.25,.32)],13)
            p.line([(.85,.53),(.92,.17),(.75,.32)],13)
            for x in [.23,.38,.53,.68]:p.oval(x,.56,.1,.13,17)

def rle(im):
    data=list(im.get_flattened_data());w,h=im.size;rows=[]
    for y in range(h):
        row=[];last=None;n=0
        for v in data[y*w:(y+1)*w]:
            if v==last:n+=1
            else:
                if last is not None:row.extend([n,last])
                last=v;n=1
        row.extend([n,last]);rows.append(row)
    return rows

def canvas(p,name,id):
    return dict(id=id,name=name,mode='half',w=p.w,h=p.h,
                anchor=dict(x=p.w//2,y=round((p.h-1)*.87)),rows=rle(p.im))

def main():
    designs=[]
    for i,name in enumerate(NAMES):
        for view,draw in [('side',side),('top',top),('front',front)]:
            lods=[]
            for size in ['HD','SD']:
                hd=size=='HD'
                if view=='side':w,h=(112,38) if hd else (56,20)
                elif view=='top':w,h=(28,56) if hd else (14,28)
                else:w,h=(50,38) if hd else (26,20)
                p=Painter(w,h);draw(p,i)
                lods.append(canvas(p,size,f'ship-{i}-{view}-{size.lower()}'))
            if view=='top':
                for size,scale in [('fleet-HD',1),('fleet-SD',.5)]:
                    w=round((12+LENGTHS[i]*2)*scale);h=round((LENGTHS[i]*9+8)*scale)
                    p=Painter(w,h);top(p,i)
                    lods.append(canvas(p,size,f'ship-{i}-{size}'))
                for size,w,step in [('board-HD',5,4),('board-SD',3,2)]:
                    p=Painter(w,LENGTHS[i]*step-1);top(p,i)
                    lods.append(canvas(p,size,f'ship-{i}-{size}'))
                # Independently rasterized horizontal tactical hulls. Author at
                # their native cell aspect, then rotate once in the asset file.
                for size,cw,ch in [('board-horizontal-HD',6,4),('board-horizontal-SD',3,2)]:
                    p=Painter(ch,LENGTHS[i]*cw-1);top(p,i)
                    p.im=p.im.transpose(Image.Transpose.ROTATE_90);p.w,p.h=p.im.size
                    lods.append(canvas(p,size,f'ship-{i}-{size}'))
            designs.append(dict(id=f'ship-{i}-{view}',name=f'{name} / {view}',lods=lods))
    # Authored concentric shock / splash frames, separate native sizes and slots.
    lods=[]
    for size,n in [('HD',35),('SD',19)]:
        p=Painter(n,n);c=canvas(p,size,'impact-'+size.lower());frames=[]
        for frame in range(8):
            p=Painter(n,n);r=(frame+1)/(8)*.46
            for a in range(0,360,14):
                ang=math.radians(a);x=.5+math.cos(ang)*r;y=.5+math.sin(ang)*r*.42
                p.line([(x,y),(x+math.cos(ang)*.04,y+math.sin(ang)*.04)],20 if frame<4 else 24)
            if frame<4:p.oval(.5-r*.3,.5-r*.3,r*.6,r*.6,21)
            frames.append(dict(hold=2,rows=rle(p.im)))
        c['anchor']={'x':n//2,'y':n//2};c['tracks']=[dict(name='burst',frames=frames)];lods.append(c)
    designs.append(dict(id='impact',name='Impact crown',lods=lods))
    doc=dict(format='dotmason',version=1,palette=PALETTE,palette_mode='fixed',
             slots={'energy':17,'foam':24,'flash':21},designs=designs)
    target=ROOT/'internal/assets/fleet.mason';target.write_text(json.dumps(doc,separators=(',',':'))+'\n')
    print(f'Authored {len(designs)} designs in {target.relative_to(ROOT)}')
if __name__=='__main__':main()
