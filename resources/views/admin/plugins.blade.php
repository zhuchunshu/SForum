<div class="alert alert-success" role="alert">
    <h4 class="alert-title">安全提示</h4>
    <div class="text-muted">如果两个插件共用同一路由,请先卸载不用的插件，以免造成冲突</div>
</div>
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
        <tbody id="vue-plugin-table">
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
