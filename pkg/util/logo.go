// +build !windows

package util

// Logo logo
var Logo = `
[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;196m [0;00m[38;5;196m [0;00m[38;5;198m [0;00m[38;5;125m [0;00m[38;5;197m [0;00m[38;5;197m [0;00m[38;5;198m [0;00m[38;5;196m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m
[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m.[0;00m[38;5;231m.[0;00m[38;5;231m,[0;00m[38;5;231m,[0;00m[38;5;231m,[0;00m[38;5;231m,[0;00m[38;5;231m,[0;00m[38;5;231m.[0;00m[38;5;231m.[0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;196m [0;00m[38;5;198m [0;00m[38;5;197m [0;00m[38;5;197m [0;00m[38;5;88m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;88m [0;00m[38;5;197m [0;00m[38;5;197m [0;00m[38;5;125m [0;00m[38;5;125m [0;00m[38;5;196m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m
[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m.[0;00m[38;5;231m:[0;00m[38;5;231mi[0;00m[38;5;231m1[0;00m[38;5;231mf[0;00m[38;5;231mC[0;00m[38;5;231mG[0;00m[38;5;231m0[0;00m[38;5;231m0[0;00m[38;5;231m8[0;00m[38;5;231m8[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m8[0;00m[38;5;231m8[0;00m[38;5;231m0[0;00m[38;5;231mG[0;00m[38;5;231mC[0;00m[38;5;231mL[0;00m[38;5;231mt[0;00m[38;5;231m;[0;00m[38;5;231m,[0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;198m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;198m [0;00m[38;5;162m.[0;00m[38;5;162m,[0;00m[38;5;162m,[0;00m[38;5;162m:[0;00m[38;5;162m:[0;00m[38;5;162m:[0;00m[38;5;162m:[0;00m[38;5;162m:[0;00m[38;5;162m:[0;00m[38;5;162m,[0;00m[38;5;162m,[0;00m[38;5;162m.[0;00m[38;5;162m.[0;00m[38;5;162m.[0;00m[38;5;161m [0;00m[38;5;162m [0;00m[38;5;198m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m
[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m,[0;00m[38;5;231m;[0;00m[38;5;231mt[0;00m[38;5;231mC[0;00m[38;5;231m0[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;224m@[0;00m[38;5;224m0[0;00m[38;5;196m:[0;00m[38;5;196m,[0;00m[38;5;22m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;198m [0;00m[38;5;162m.[0;00m[38;5;162m,[0;00m[38;5;162m:[0;00m[38;5;162mi[0;00m[38;5;162mi[0;00m[38;5;162m1[0;00m[38;5;162m1[0;00m[38;5;162m1[0;00m[38;5;162mi[0;00m[38;5;162m;[0;00m[38;5;162m:[0;00m[38;5;162m,[0;00m[38;5;162m,[0;00m[38;5;162m.[0;00m[38;5;162m.[0;00m[38;5;162m [0;00m[38;5;197m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m
[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m,[0;00m[38;5;231m;[0;00m[38;5;231m1[0;00m[38;5;231mt[0;00m[38;5;231mt[0;00m[38;5;231mt[0;00m[38;5;231m1[0;00m[38;5;231m;[0;00m[38;5;231m;[0;00m[38;5;231m:[0;00m[38;5;231m:[0;00m[38;5;231m:[0;00m[38;5;231m:[0;00m[38;5;231m;[0;00m[38;5;231m;[0;00m[38;5;231mi[0;00m[38;5;231mt[0;00m[38;5;231mf[0;00m[38;5;231mC[0;00m[38;5;231mG[0;00m[38;5;231m8[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;224m@[0;00m[38;5;224m@[0;00m[38;5;224m@[0;00m[38;5;224m@[0;00m[38;5;224m@[0;00m[38;5;196mi[0;00m[38;5;16m [0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;225m@[0;00m[38;5;182m0[0;00m[38;5;175mf[0;00m[38;5;162mi[0;00m[38;5;162m1[0;00m[38;5;162mt[0;00m[38;5;161m1[0;00m[38;5;161m1[0;00m[38;5;162m1[0;00m[38;5;162mi[0;00m[38;5;162m;[0;00m[38;5;162m,[0;00m[38;5;162m.[0;00m[38;5;204m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m
[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m.[0;00m[38;5;231m,[0;00m[38;5;231m.[0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;224m.[0;00m[38;5;224m:[0;00m[38;5;224m1[0;00m[38;5;224mL[0;00m[38;5;224m0[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;188m@[0;00m[38;5;188m8[0;00m[38;5;181mG[0;00m[38;5;175mC[0;00m[38;5;168mf[0;00m[38;5;162mt[0;00m[38;5;161m1[0;00m[38;5;161m1[0;00m[38;5;161m1[0;00m[38;5;162mi[0;00m[38;5;168mi[0;00m[38;5;175m1[0;00m[38;5;182m;[0;00m[38;5;30m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;132m [0;00m[38;5;211m [0;00m[38;5;175m [0;00m[38;5;218m [0;00m[38;5;219m [0;00m[38;5;217m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m
[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;217m [0;00m[38;5;217m [0;00m[38;5;218m [0;00m[38;5;211m [0;00m[38;5;175m [0;00m[38;5;218m [0;00m[38;5;196m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;224m:[0;00m[38;5;188mf[0;00m[38;5;182m0[0;00m[38;5;175mC[0;00m[38;5;168mL[0;00m[38;5;198mf[0;00m[38;5;161mt[0;00m[38;5;161m1[0;00m[38;5;161mi[0;00m[38;5;161m;[0;00m[38;5;162mi[0;00m[38;5;175m1[0;00m[38;5;188mt[0;00m[38;5;225mL[0;00m[38;5;16m [0;00m[38;5;16m.[0;00m[38;5;16m [0;00m[38;5;231m@[0;00m[38;5;231m8[0;00m[38;5;224mC[0;00m[38;5;224mt[0;00m[38;5;188mi[0;00m[38;5;224m:[0;00m[38;5;231m.[0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m.[0;00m[38;5;231m,[0;00m[38;5;231m:[0;00m[38;5;231m,[0;00m[38;5;231m.[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m
[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;196m [0;00m[38;5;196m [0;00m[38;5;196m [0;00m[38;5;196m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;162m [0;00m[38;5;162m.[0;00m[38;5;162m:[0;00m[38;5;162m;[0;00m[38;5;162mi[0;00m[38;5;161m1[0;00m[38;5;161m1[0;00m[38;5;161m1[0;00m[38;5;161m1[0;00m[38;5;161mi[0;00m[38;5;162m;[0;00m[38;5;168mi[0;00m[38;5;182mt[0;00m[38;5;224mC[0;00m[38;5;196m;[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;225m@[0;00m[38;5;224m@[0;00m[38;5;224m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m8[0;00m[38;5;231m0[0;00m[38;5;231mG[0;00m[38;5;231mC[0;00m[38;5;231mL[0;00m[38;5;231mf[0;00m[38;5;231mt[0;00m[38;5;231mt[0;00m[38;5;231m1[0;00m[38;5;231m1[0;00m[38;5;231mt[0;00m[38;5;231mt[0;00m[38;5;231mf[0;00m[38;5;231mL[0;00m[38;5;231mL[0;00m[38;5;231mL[0;00m[38;5;231mf[0;00m[38;5;231m1[0;00m[38;5;231m:[0;00m[38;5;231m.[0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m
[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;198m [0;00m[38;5;162m [0;00m[38;5;162m.[0;00m[38;5;162m,[0;00m[38;5;162m:[0;00m[38;5;162m;[0;00m[38;5;162mi[0;00m[38;5;162mi[0;00m[38;5;162m1[0;00m[38;5;162m1[0;00m[38;5;162m1[0;00m[38;5;162m1[0;00m[38;5;162mi[0;00m[38;5;162m:[0;00m[38;5;162m,[0;00m[38;5;162m.[0;00m[38;5;211m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;33m [0;00m[38;5;196m,[0;00m[38;5;16m [0;00m[38;5;224mG[0;00m[38;5;224m8[0;00m[38;5;224m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m@[0;00m[38;5;231m0[0;00m[38;5;231mL[0;00m[38;5;231mt[0;00m[38;5;231m;[0;00m[38;5;231m,[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m
[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;125m [0;00m[38;5;197m [0;00m[38;5;162m [0;00m[38;5;162m.[0;00m[38;5;161m.[0;00m[38;5;162m.[0;00m[38;5;162m,[0;00m[38;5;162m,[0;00m[38;5;162m:[0;00m[38;5;162m:[0;00m[38;5;162m;[0;00m[38;5;162m;[0;00m[38;5;162m;[0;00m[38;5;162m;[0;00m[38;5;162m:[0;00m[38;5;162m:[0;00m[38;5;162m,[0;00m[38;5;162m.[0;00m[38;5;198m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;196m [0;00m[38;5;197m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m,[0;00m[38;5;231m;[0;00m[38;5;231mi[0;00m[38;5;231mt[0;00m[38;5;231mf[0;00m[38;5;231mL[0;00m[38;5;231mC[0;00m[38;5;231mG[0;00m[38;5;231mG[0;00m[38;5;231mG[0;00m[38;5;231m0[0;00m[38;5;231mG[0;00m[38;5;231mG[0;00m[38;5;231mG[0;00m[38;5;231mC[0;00m[38;5;231mL[0;00m[38;5;231mf[0;00m[38;5;231mt[0;00m[38;5;231m1[0;00m[38;5;231m;[0;00m[38;5;231m:[0;00m[38;5;231m.[0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m
[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;196m [0;00m[38;5;197m [0;00m[38;5;198m [0;00m[38;5;162m [0;00m[38;5;162m [0;00m[38;5;161m [0;00m[38;5;161m [0;00m[38;5;198m [0;00m[38;5;197m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;196m [0;00m[38;5;197m [0;00m[38;5;197m [0;00m[38;5;198m [0;00m[38;5;196m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m
[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;196m [0;00m[38;5;203m [0;00m[38;5;211m [0;00m[38;5;211m [0;00m[38;5;211m [0;00m[38;5;175m [0;00m[38;5;218m [0;00m[38;5;217m [0;00m[38;5;145m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m
[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;217m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m
[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m;[0;00m[38;5;231m1[0;00m[38;5;231m:[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231mi[0;00m[38;5;231m1[0;00m[38;5;231m,[0;00m[38;5;16m [0;00m[38;5;231m.[0;00m[38;5;231m1[0;00m[38;5;231m,[0;00m[38;5;16m [0;00m[38;5;231m.[0;00m[38;5;231m1[0;00m[38;5;231m1[0;00m[38;5;231m1[0;00m[38;5;231m1[0;00m[38;5;231m1[0;00m[38;5;231mi[0;00m[38;5;231m:[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231m.[0;00m[38;5;231m1[0;00m[38;5;231mi[0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231m:[0;00m[38;5;231m1[0;00m[38;5;231m;[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231m,[0;00m[38;5;231m1[0;00m[38;5;231m,[0;00m[38;5;16m [0;00m[38;5;231mi[0;00m[38;5;231m1[0;00m[38;5;231m1[0;00m[38;5;231m1[0;00m[38;5;231m1[0;00m[38;5;231m1[0;00m[38;5;231m1[0;00m[38;5;231m1[0;00m[38;5;231m1[0;00m[38;5;231m:[0;00m[38;5;16m [0;00m[38;5;231m;[0;00m[38;5;231mi[0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m;[0;00m[38;5;231m1[0;00m[38;5;231m1[0;00m[38;5;231m1[0;00m[38;5;231m1[0;00m[38;5;231m;[0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m
[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231mG[0;00m[38;5;231m@[0;00m[38;5;231mG[0;00m[38;5;231mC[0;00m[38;5;231m:[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m1[0;00m[38;5;231mG[0;00m[38;5;231m0[0;00m[38;5;231m@[0;00m[38;5;231mi[0;00m[38;5;16m [0;00m[38;5;231m;[0;00m[38;5;231m@[0;00m[38;5;231m1[0;00m[38;5;16m [0;00m[38;5;231m;[0;00m[38;5;231m@[0;00m[38;5;231mt[0;00m[38;5;231m:[0;00m[38;5;231m:[0;00m[38;5;231m:[0;00m[38;5;231m;[0;00m[38;5;231mG[0;00m[38;5;231m0[0;00m[38;5;231m.[0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231m.[0;00m[38;5;231m0[0;00m[38;5;231mG[0;00m[38;5;231m8[0;00m[38;5;231mf[0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231mL[0;00m[38;5;231m@[0;00m[38;5;231mG[0;00m[38;5;231mC[0;00m[38;5;231m:[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231mi[0;00m[38;5;231m@[0;00m[38;5;231mi[0;00m[38;5;16m [0;00m[38;5;231m:[0;00m[38;5;231m;[0;00m[38;5;231m;[0;00m[38;5;231m;[0;00m[38;5;231m@[0;00m[38;5;231mG[0;00m[38;5;231m;[0;00m[38;5;231m;[0;00m[38;5;231m;[0;00m[38;5;231m,[0;00m[38;5;16m [0;00m[38;5;231mC[0;00m[38;5;231m@[0;00m[38;5;231m [0;00m[38;5;231m.[0;00m[38;5;231m8[0;00m[38;5;231m0[0;00m[38;5;231mi[0;00m[38;5;231m:[0;00m[38;5;231m,[0;00m[38;5;231m:[0;00m[38;5;231m;[0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m
[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231mC[0;00m[38;5;231m8[0;00m[38;5;231m [0;00m[38;5;231m1[0;00m[38;5;231m0[0;00m[38;5;231mC[0;00m[38;5;231mG[0;00m[38;5;231mC[0;00m[38;5;231m:[0;00m[38;5;231m:[0;00m[38;5;231m@[0;00m[38;5;231mi[0;00m[38;5;16m [0;00m[38;5;231m:[0;00m[38;5;231m@[0;00m[38;5;231mi[0;00m[38;5;16m [0;00m[38;5;231m:[0;00m[38;5;231m@[0;00m[38;5;231mf[0;00m[38;5;231m:[0;00m[38;5;231m;[0;00m[38;5;231m;[0;00m[38;5;231m;[0;00m[38;5;231mC[0;00m[38;5;231mG[0;00m[38;5;231m.[0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231m.[0;00m[38;5;231m0[0;00m[38;5;231mG[0;00m[38;5;16m [0;00m[38;5;231m,[0;00m[38;5;231m@[0;00m[38;5;231mL[0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231mf[0;00m[38;5;231m@[0;00m[38;5;231m.[0;00m[38;5;231mi[0;00m[38;5;231mG[0;00m[38;5;231mL[0;00m[38;5;231m,[0;00m[38;5;16m [0;00m[38;5;231m;[0;00m[38;5;231m@[0;00m[38;5;231m;[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m@[0;00m[38;5;231mL[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231mL[0;00m[38;5;231m@[0;00m[38;5;231m.[0;00m[38;5;231m [0;00m[38;5;231mt[0;00m[38;5;231mL[0;00m[38;5;231mf[0;00m[38;5;231mt[0;00m[38;5;231m1[0;00m[38;5;231m;[0;00m[38;5;231m.[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m
[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231mG[0;00m[38;5;231m8[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m1[0;00m[38;5;231m;[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m;[0;00m[38;5;231m@[0;00m[38;5;231mi[0;00m[38;5;16m [0;00m[38;5;231m;[0;00m[38;5;231m@[0;00m[38;5;231mi[0;00m[38;5;16m [0;00m[38;5;231m:[0;00m[38;5;231m@[0;00m[38;5;231mL[0;00m[38;5;231m1[0;00m[38;5;231m1[0;00m[38;5;231mC[0;00m[38;5;231m@[0;00m[38;5;231mt[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m,[0;00m[38;5;231m0[0;00m[38;5;231m8[0;00m[38;5;231mf[0;00m[38;5;231mf[0;00m[38;5;231mf[0;00m[38;5;231mL[0;00m[38;5;231m@[0;00m[38;5;231mL[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231mL[0;00m[38;5;231m@[0;00m[38;5;231m.[0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m1[0;00m[38;5;231m0[0;00m[38;5;231mf[0;00m[38;5;231mi[0;00m[38;5;231m@[0;00m[38;5;231m;[0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m.[0;00m[38;5;231m@[0;00m[38;5;231mL[0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231mL[0;00m[38;5;231m@[0;00m[38;5;231m.[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m.[0;00m[38;5;231m:[0;00m[38;5;231mt[0;00m[38;5;231m@[0;00m[38;5;231m1[0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m
[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231mC[0;00m[38;5;231m0[0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m;[0;00m[38;5;231m@[0;00m[38;5;231mi[0;00m[38;5;16m [0;00m[38;5;231m:[0;00m[38;5;231m@[0;00m[38;5;231mi[0;00m[38;5;16m [0;00m[38;5;231m:[0;00m[38;5;231m@[0;00m[38;5;231mi[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m;[0;00m[38;5;231mG[0;00m[38;5;231mf[0;00m[38;5;231m.[0;00m[38;5;231m.[0;00m[38;5;231m0[0;00m[38;5;231mC[0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231m,[0;00m[38;5;231m8[0;00m[38;5;231mL[0;00m[38;5;231m [0;00m[38;5;231mf[0;00m[38;5;231m@[0;00m[38;5;231m.[0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m.[0;00m[38;5;231mf[0;00m[38;5;231m0[0;00m[38;5;231m@[0;00m[38;5;231m;[0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m.[0;00m[38;5;231m8[0;00m[38;5;231mL[0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231mL[0;00m[38;5;231m8[0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231mf[0;00m[38;5;231mf[0;00m[38;5;231mt[0;00m[38;5;231m1[0;00m[38;5;231m1[0;00m[38;5;231mf[0;00m[38;5;231mG[0;00m[38;5;231m;[0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m
[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m.[0;00m[38;5;231m.[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m.[0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m.[0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m.[0;00m[38;5;231m.[0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m,[0;00m[38;5;231m.[0;00m[38;5;231m.[0;00m[38;5;231m,[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m,[0;00m[38;5;231m [0;00m[38;5;231m.[0;00m[38;5;231m.[0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m.[0;00m[38;5;231m.[0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m.[0;00m[38;5;231m.[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m.[0;00m[38;5;231m.[0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;231m.[0;00m[38;5;231m,[0;00m[38;5;231m:[0;00m[38;5;231m:[0;00m[38;5;231m:[0;00m[38;5;231m,[0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m
[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;231m [0;00m[38;5;231m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m[38;5;16m [0;00m
`
