import json
try:
    import commands
except ImportError:
    import subprocess as commands
import os
f = open('config.json')
configs = json.load(f)
for ele in configs['Configs']:
    cmd = 'nohup shadowsocks-local -s {} -p {} -k {} -l {} -m {} &'.format(
        ele['Server'], ele['RemotePort'], ele['PassWord'], ele['LocalPort'], ele['Method'])
    print(cmd)
    commands.Popen(cmd,shell=True)
