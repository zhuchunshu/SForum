@extends('app')
@section('title','待更新插件')

@section('content')
    <div class="col-12">
        <div class="card" id="admin-plugin-newVersion">
            <div class="card-header">
                <h3 class="card-title">待更新插件</h3>
            </div>
            <div class="card-table table-responsive">
                <table class="table table-vcenter">
                    <thead>
                    <tr>
                        <th>插件</th>
                        <th>插件id</th>
                        <th>当前版本</th>
                        <th>最新版本</th>
                        <th class="w-8"></th>
                    </tr>
                    </thead>
                    <tbody v-if="plugins===null">
                    <tr>
                        <td colspan="5" class="text-muted text-center">
                            加载中...
                        </td>
                    </tr>
                    </tbody>
                    <tbody v-else-if="pluginsArrayLengthGreaterThanZero">
                    <tr v-for="plugin in plugins">
                        <td>@{{ plugin.name }}</td>
                        <td><a target="_blank" :href="plugin.url">@{{ plugin.aid }}</a> </td>
                        <td>@{{ plugin.version }}</td>
                        <td>@{{ plugin.new_version }}</td>
                        <td><a target="_blank" :href="plugin.download" class="btn btn-link">前往下载</a> </td>
                    </tbody>
                    <tbody v-else>
                    <tr>
                        <td colspan="4" class="text-muted text-center">
                            暂无可更新插件
                        </td>
                    </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>
@endsection

@section('scripts')
    <script>
        const app = {
            data() {
                return {
                    plugins: null
                }
            },
            mounted() {
                // fetch 发送 post请求
                fetch('/admin/plugins/newVersion', {
                    method: 'post',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        _token: '{{csrf_token()}}'
                    })
                }).then(res => res.json()).then(res => {
                    if(res.success===true){
                        this.plugins = res.result
                    }
                })
            },
            computed: {
                pluginsArrayLengthGreaterThanZero() {
                    return this.plugins!==null && this.plugins.length > 0;
                }
            },
        }
        Vue.createApp(app).mount('#admin-plugin-newVersion')
    </script>
@endsection