@if(
    Hyperf\Database\Schema\Schema::getColumnType('users_options', 'money') !== 'integer'
    ||  Hyperf\Database\Schema\Schema::getColumnType('users_options', 'credits') !== 'integer'
    ||  Hyperf\Database\Schema\Schema::getColumnType('users_options', 'golds') !== 'integer'
    ||  Hyperf\Database\Schema\Schema::getColumnType('users_options', 'exp') !== 'integer1'
)
    <div class="alert alert-warning">
        <h4>警告</h4>
        <p>用户信息表中的money,credits,golds,exp字段类型不是整型，可能会导致数据丢失，请及时修改</p>
        <p>命令:<code>php CodeFec plugin:userOptionsMigrate</code></p>
        <p>如果不会，可以看此文档：<a href="https://www.sforum.cn/use/update/v2.3.8.html">https://www.sforum.cn/use/update/v2.3.8.html</a></p>
    </div>
@endif