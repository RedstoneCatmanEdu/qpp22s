import logging
import sys
import os
import argparse
from os_ken.base import app_manager
from os_ken import cfg as os_ken_cfg

if __name__ == '__main__':
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s [%(levelname)s] %(name)s: %(message)s',
        stream=sys.stdout
    )
    
    parser = argparse.ArgumentParser(add_help=False)
    parser.add_argument('-t', '--idle-timeout', type=int, default=60, help='Idle timeout for flows')
    args, unknown = parser.parse_known_args()
    
    os.environ['FLOW_IDLE_TIMEOUT'] = str(args.idle_timeout)
    os_ken_cfg.CONF(project='os_ken', args=unknown)
    
    app_manager.AppManager.run_apps([
        'controler_app'
    ])