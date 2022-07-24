let curatedStyles = [
  require("./base16/base16-catppuccin.js"),
  require("./base16/base16-chalk.js"),
  require("./base16/base16-classic-dark.js"),
  require("./base16/base16-darkmoss.js"),
  require("./base16/base16-decaf.js"),
  require("./base16/base16-default-dark.js"),
  require("./base16/base16-dracula.js"),
  require("./base16/base16-eighties.js"),
  require("./base16/base16-equilibrium-dark.js"),
  require("./base16/base16-equilibrium-gray-dark.js"),
  require("./base16/base16-espresso.js"),
  require("./base16/base16-framer.js"),
  require("./base16/base16-gruvbox-dark-medium.js"),
  require("./base16/base16-hardcore.js"),
  require("./base16/base16-horizon-dark.js"),
  require("./base16/base16-horizon-terminal-dark.js"),
  require("./base16/base16-irblack.js"),
  require("./base16/base16-materia.js"),
  require("./base16/base16-material.js"),
  require("./base16/base16-mocha.js"),
  require("./base16/base16-monokai.js"),
  require("./base16/base16-ocean.js"),
  require("./base16/base16-oceanicnext.js"),
  require("./base16/base16-outrun-dark.js"),
  require("./base16/base16-rose-pine.js"),
  require("./base16/base16-rose-pine-moon.js"),
  require("./base16/base16-snazzy.js"),
  require("./base16/base16-tender.js"),
  require("./base16/base16-tomorrow-night.js"),
  require("./base16/base16-tomorrow-night-eighties.js"),
  require("./base16/base16-twilight.js"),
  require("./base16/base16-woodland.js"),
];

// NB: this has to be deterministic so multiple renders don't switch styles
let choice = new Date().getMinutes();

export let style = curatedStyles[choice % curatedStyles.length];
