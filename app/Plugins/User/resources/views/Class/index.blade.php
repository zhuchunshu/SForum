@extends("app")

@section('title',"用户组管理")

@section('headerBtn')
    <a href="/admin/userClass/create" class="btn btn-primary">创建用户组</a>
@endsection

@section('content')
    <div class="col-md-12">
        <div class="card card-body">
            <div class="table-responsive">
                <table class="table table-vcenter table-nowrap">
                    <thead>
                    <tr>
                        <th>#</th>
                        <th>名称</th>
                        <th>标识</th>
                        <th>颜色</th>
                        <th>权限</th>
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
        </div>
    </div>
@endsection
