import {Config} from '@remotion/cli/config';

// Use the host's Chrome rather than downloading Remotion's headless shell.
Config.setBrowserExecutable('/usr/bin/google-chrome');
Config.setVideoImageFormat('jpeg');
Config.setOverwriteOutput(true);
