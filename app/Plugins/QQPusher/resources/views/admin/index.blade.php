@extends('app')
@section('title','QQPusher设置')
@section('content')

    <div class="row row-cards">
        <div class="col-md-12">

            <div class="card card-body">
                <h3 class="card-title">GO_CQHTTP配置</h3>
                <small> <a href="https://docs.go-cqhttp.org/guide/quick_start.html#%E5%9F%BA%E7%A1%80%E6%95%99%E7%A8%8B">https://docs.go-cqhttp.org/guide/quick_start.html</a> </small>
                <form action="/admin/QQPusher/save" method="post">
                    <x-csrf/>
                    <div class="mb-3">
                        <label for="" class="form-label">
                            POST请求地址
                        </label>
                        <input name="QQPusher_POST_URL" required type="url" class="form-control" value="{{get_options('QQPusher_POST_URL','http://127.0.0.1:5700')}}">
                    </div>
                    <div class="mb-3">
                        <button class="btn btn-primary">保存</button>
                    </div>
                </form>
            </div>

        </div>

        <div class="col-md-12">
            <div class="card card-body">
                <h3 class="card-title">QQ群</h3>
                <div class="table-responsive" id="groups">
                    <table
                            class="table table-vcenter">
                        <thead>
                        <tr>
                            <th class="w-1"></th>
                            <th>群名</th>
                            <th>群号</th>
                            <th class="w-1"></th>
                        </tr>
                        </thead>
                        <tbody>

                        @if(count($groups))
                            @foreach($groups as $group)
                                <tr>
                                    <td> <span class="avatar avatar-rounded avatar-sm" style="background-image: url(https://p.qlogo.cn/gh/{{$group['group_id']}}/{{$group['group_id']}}/100)"></span> </td>
                                    <td class="text-muted" >
                                        {{$group['group_name']}}
                                    </td>
                                    <td class="text-muted" >{{$group['group_id']}}</td>
                                    <td>
                                        <label class="form-check form-switch"><input v-model="checkeds" value="{{$group['group_id']}}" class="form-check-input" type="checkbox"></label>
                                    </td>
                                </tr>
                            @endforeach
                        @else
                            <tr>
                                <td ></td>
                                <td class="text-muted" >
                                    暂无
                                </td>
                                <td class="text-muted" >暂无</td>
                                <td ></td>
                            </tr>
                        @endif
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    </div>

@endsection

@section('scripts')
    <script src="{{file_hash('plugins/QQPusher/js/admin.js')}}"></script>
@endsection