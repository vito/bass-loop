const styleKey = "theme";
const linkId = "theme";
const iconId = "favicon";

const defaultId = "default-theme"
const controlsId = "choosetheme"
const switcherId = "styleswitcher";
const resetId = "resetstyle";

function storeStyle(style) {
  window.localStorage.setItem(styleKey, style);
}

function loadStyle() {
  var style = window.localStorage.getItem(styleKey);

  if (style !== null && window.goatcounter !== undefined && window.goatcounter.count !== undefined) {
    window.goatcounter.count({
      path:  `style/${style}`,
      title: "Loaded user-configured style",
      event: true,
    })
  }

  return style
}

function setActiveStyle(style) {
  var link = document.createElement('link');
  link.rel = "stylesheet";
  link.type = "text/css";
  link.href = "/css/base16/base16-"+style+".css";
  link.media = "all";

  // only swap the element once the CSS has loaded to prevent flickering
  link.onload = function() {
    // switcheroo
    var prevLink = document.getElementById(linkId)
    if (prevLink) {
      prevLink.remove();
    }

    link.id = linkId;

    setActiveIcon(style);
  };

  document.head.appendChild(link);

  var switcher = document.getElementById(switcherId);
  if (switcher) {
    // might not be loaded yet; this function is called twice, once super early
    // to prevent flickering, and again once all the dom is loaded up
    switcher.value = style;
  }

  var defaultStyle = document.getElementById(defaultId);
  if (defaultStyle) {
    defaultStyle.disabled = true;
  }

  resetReset();
}

function setActiveIcon(style) {
  var link = document.createElement('link');
  link.rel = "shortcut icon";
  link.type = "image/x-icon";
  link.href = "/ico/base16-"+style+".svg";

  // switcheroo
  var prevLink = document.getElementById(iconId)
  if (prevLink) {
    prevLink.remove();
  }

  link.id = iconId;

  document.head.appendChild(link);
}

function resetReset() {
  var style = loadStyle();
  var reset = document.getElementById(resetId);
  if (!style) {
    if (reset) {
      // no style selected; remove reset element
      reset.remove();
    }

    return
  }

  if (reset) {
    // no style and no reset; done
    return
  }

  // has style but no reset element
  reset = document.createElement("a");
  reset.id = resetId;
  reset.onclick = resetStyle;
  reset.href = 'javascript:void(0)';
  reset.text = "reset";
  reset.className = "reset";

  var chooser = document.getElementById(controlsId);
  if (chooser) {
    chooser.appendChild(reset);
  }
}

export function setStyleIfSet() {
  var style = loadStyle();
  if (!style) {
    return;
  }

  setActiveStyle(style);
}

function resetStyle() {
  window.localStorage.removeItem(styleKey);

  var linkStyle = document.getElementById(linkId)
  if (linkStyle) {
    linkStyle.remove();
  }

  var defaultStyle = document.getElementById(defaultId);
  if (defaultStyle) {
    defaultStyle.disabled = false;

    var switcher = document.getElementById(switcherId);
    if (switcher) {
      switcher.value = switcher.dataset.default;
    }
  }

  resetReset();
}

setStyleIfSet();

window.onload = function() {
  setStyleIfSet();
}

export function switchStyle(event) {
  var style = event.target.value;
  storeStyle(style);
  setActiveStyle(style);
}
