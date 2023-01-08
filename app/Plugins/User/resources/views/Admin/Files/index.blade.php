@extends("app")

@section('title',"文件管理")


@section('content')
    <div class="col-md-12">
        <div class="border-0 card">
            <div class="card-body">
                <h3 class="card-title">文件管理</h3>
                <div class="row">
                    <div class="col"></div>

                </div>
                @if($page->count())
                    <div class="table-responsive" id="vue-users-files">
                        <table
                                class="table table-vcenter table-nowrap">
                            <thead>
                            <tr>
                                <th>#</th>
                                <th>类型</th>
                                <th>文件名</th>
                                <th>创建者</th>
                                <th>路径</th>
                                <th>url</th>
                                <th>创建时间</th>
                                <th class="w-1"></th>
                                <th class="w-1"></th>
                            </tr>
                            </thead>
                            <tbody>
                            @foreach($page as $data)
                                <tr>
                                    <td>{{$data->id}}</td>
                                    <td>
                                        <span class="avatar bg-blue-lt">{{file_suffix($data->path)}}</span>
                                    </td>
                                    <td @@click="alert('{{path_file_name($data->path)}}')" class="text-truncate" style="max-width: 100px">
                                        {{path_file_name($data->path)}}
                                    </td>
                                    <td>
                                        <a href="/users/{{$data->user->id}}.html"><span class="avatar" style="background-image: url({{super_avatar($data->user)}})"></span></a>
                                    </td>
                                    <td  @@click="alert('{{$data->path}}')" data-bs-toggle="tooltip" data-bs-placement="top" title="{{$data->path}}" class="text-truncate" style="max-width: 100px">{{$data->path}}</td>
                                    <td class="text-truncate" style="max-width: 100px"><a href="{{$data->url}}">{{$data->url}}</a></td>
                                    <td>{{$data->created_at}}</td>
                                    <td><a @@click="download('{{$data->url}}')" href="#">下载</a></td>
                                    <td><a @@click="remove({{$data->id}})" href="#">删除</a></td>
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