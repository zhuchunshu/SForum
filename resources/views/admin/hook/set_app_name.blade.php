@if(!get_options('APP_NAME') || !get_options('web_name'))
    <div class="my-2">
        <div class="alert alert-danger m-0">
            请先<a href="/admin/setting" class="alert-link">--->设置(点我)<---</a>网站名和应用名、否则会导致部分功能无法正常运行，例如：邮箱发信
        </div>
    </div>

@endif