let mix = require('laravel-mix');

function public_path($path) {
    if ($path) {
        return "./public/" + $path;
    } else {
        return "./public"
    }
}

function resources_path($path) {
    if ($path) {
        return "./resources/" + $path;
    } else {
        return "./resources"
    }
}

// app.js
mix.js(resources_path("js/app.js"), "js").version();
// install.js
mix.js(resources_path("js/install.js"), "js").version();


//admin
// login
mix.js(resources_path("js/admin/login.js"), "js/admin").version();

//EditFile
mix.js(resources_path("js/admin/EditFile.js"), "js/admin").version();
// error
mix.js(resources_path("js/admin/error.js"), "js/admin").version();
// component
mix.js(resources_path("js/admin/component.js"), "js/admin").version();
// setting
mix.js(resources_path("js/admin/setting.js"), "js/admin").version();
mix.js(resources_path("js/admin/index.js"), "js/admin").version();
// pay
mix.js(resources_path("js/admin/pay.js"), "js/admin").version();
try {
    require("./plugins.mix")
} catch {

}

mix.css(resources_path("sass/app.css"), "css").version();


// 设置public目录
mix.setPublicPath(public_path());

mix.setResourceRoot(resources_path());

