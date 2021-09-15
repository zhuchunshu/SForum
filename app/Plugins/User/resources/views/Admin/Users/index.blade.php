@extends("app")

@section('title',"用户列表")


@section('content')
    <div class="col-md-12">
        <div class="border-0 card">
            <div class="card-body">
                <h3 class="card-title">用户列表</h3>
                @if($page->count())
                    <div class="table-responsive" id="vue-users">
                        <table
                                class="table table-vcenter table-nowrap">
                            <thead>
                            <tr>
                                <th>#</th>
                                <th>头像</th>
                                <th>用户名</th>
                                <th>邮箱</th>
                                <th>用户组</th>
                                <th>注册时间</th>
                                <th>最后更新时间</th>
                                <th>Token</th>
                                <th class="w-1"></th>
                                <th class="w-1"></th>
                            </tr>
                            </thead>
                            <tbody>
                            @foreach($page as $data)
                                <tr>
                                    <td>{{$data->id}}</td>
                                    <td>
                                        <span class="avatar avatar-sm" style="background-image: url({{super_avatar($data)}})"></span>
                                    </td>
                                    <td @@click="username('{{$data->id}}','{{$data->username}}')">
                                        {{$data->username}}
                                    </td>
                                    <td @@click="email('{{$data->id}}','{{$data->email}}')">
                                        {{$data->email}}
                                    </td>
                                    <td @@click="UserClass('{{$data->id}}','{{$data->class->id}}')">
                                        {{$data->class->name}}
                                    </td>
                                    <td>
                                        {{$data->created_at}}
                                    </td>
                                    <td>
                                        {{$data->updated_at}}
                                    </td>
                                    <td @@click="token('{{$data->id}}','{{$data->_token}}')">
                                        {{$data->_token}}
                                    </td>
                                    <td>
                                        <a @@click="re_pwd('{{$data->id}}')" href="#">重置密码</a>
                                    </td>
                                    <td>
                                        <a @@click="remove('{{$data->id}}')" href="#">删除</a>
                                    </td>
                                </tr>
                            @endforeach
                            </tbody>
                        </table>
                    </div>
                @else
                    无用户
                @endif
            </div>
            {!! make_page($page) !!}
        </div>
    </div>
@endsection

@section('scripts')
    <script src="{{ mix("plugins/User/js/user.js") }}"></script>
@endsection