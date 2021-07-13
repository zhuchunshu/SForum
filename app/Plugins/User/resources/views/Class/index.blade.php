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
                        <th>颜色</th>
                        <th>权限</th>
                        <th class="w-1"></th>
                        <th class="w-1"></th>
                    </tr>
                    </thead>
                    <tbody id="vue-user-class-table">
                    @foreach ($page as $value)
                        <tr>
                            <td>{{ $value->id }}</td>
                            <td>{{ $value->name }}</td>
                            <td class="text-muted">
                                <div style="width: 25px;height:25px;background-color:{{ $value->color }};border-radius:5px;"></div>
                            </td>
                            <td class="text-muted">{{ $value->quanxian }}</td>
                            <td>
                                <a href="/admin/userClass/edit/{{ $value->id }}">修改</a>
                            </td>
                            <td>
                                <a @@click="remove('{{ $value['dir'] }}','{{ $value['path'] }}')" href="#">删除</a>
                            </td>
                        </tr>
                    @endforeach
                    </tbody>
                </table>
            </div>
            {!! make_page($page) !!}
        </div>
    </div>
@endsection
