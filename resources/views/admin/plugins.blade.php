<a @@click="migrateAll" href="#">对所有已启动插件进行数据迁移</a>,
<a @@click="updatePluginsPackage" href="#">更新插件包</a>
<div class="table-responsive">
    <table class="table table-vcenter table-nowrap">
        <thead>
        <tr>
            <th>插件目录</th>
            <th>插件名</th>
            <th>作者</th>
            <th>插件版本</th>
            <th>插件描述</th>
            <th class="w-1"></th>
            <th class="w-1"></th>
            <th class="w-1"></th>
        </tr>
        </thead>
        <tbody>
        @foreach (\App\CodeFec\Plugins::GetAll() as $key => $value)
            <tr>
                <td>{{ '/app/Plugins/' . $key }}</td>
                <td>{{ $value['data']['name'] }}</td>
                <td class="text-muted">
                    <a href="{{ $value['data']['link'] }}">{{ $value['data']['author'] }}</a>
                </td>
                <td class="text-muted">{{ $value['data']['version'] }}</td>
                <td class="text-muted">{{ $value['data']['package'] }}</td>
                <td>
                    <label class="form-check form-switch">
                        <input class="form-check-input" value="{{ $value['dir'] }}" type="checkbox"
                               v-model="switchs">
                    </label>
                </td>
                <td>
                    <a @@click="migrate('{{ $value['dir'] }}','{{ $value['path'] }}')" href="#">数据迁移</a>
                </td>
                <td>
                    <a @@click="remove('{{ $value['dir'] }}','{{ $value['path'] }}')" href="#">卸载</a>
                </td>
            </tr>
        @endforeach
        </tbody>
    </table>
</div>
