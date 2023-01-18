@extends("app")

@section('title',"文件管理")


@section('content')
    <div class="col-md-12">
        <div class="border-0 card">
            <div class="card-body">
                <h3 class="card-title">用户文件管理</h3>
                <div class="row">
                    <div class="col"></div>

                </div>

                <div class="table-responsive" id="vue-users-files">
                    <table
                            class="table table-vcenter table-nowrap">
                        <thead>
                        <tr>
                            <th>#</th>
                            <th>类型</th>
                            <th>路径</th>
                            <th>文件名</th>
                            <th>url</th>
                            <th>创建时间</th>
                            <th class="w-1">文件大小</th>
                            <th class="w-1"></th>
                        </tr>
                        </thead>
                        <tbody>
                        @if($page->count())
                            @foreach($page as $file)
                                <tr>
                                    <td>
                                        {{$file['id']}}
                                    </td>
                                    <td>
                                        <span class="avatar bg-blue-lt">{{$file['extension']}}</span>
                                    </td>
                                    <td @@click="alert('{{$file['path']}}')" class="text-truncate"
                                        style="max-width: 100px">
                                        {{($file['path'])}}
                                    </td>
                                    <td @@click="alert('{{$file['filename']}}')" data-bs-toggle="tooltip"
                                        data-bs-placement="top" title="{{$file['filename']}}" class="text-truncate"
                                        style="max-width: 100px">{{$file['filename']}}</td>
                                    <td class="text-truncate" style="max-width: 100px"><a
                                                href="{{$file['url']}}">{{$file['url']}}</a></td>
                                    <td>{{$file['date']}}</td>
                                    <td>{{round($file['size']/1024,2)}}KB</td>
                                    <td><a @@click="download('{{$file['url']}}')" href="#">下载</a></td>
                                </tr>
                            @endforeach
                        @else
                            <tr>
                                <td>无更多结果</td>
                                <td>无更多结果</td>
                                <td>无更多结果</td>
                                <td>无更多结果</td>
                                <td>无更多结果</td>
                                <td>无更多结果</td>
                                <td>无更多结果</td>
                                <td>无更多结果</td>
                            </tr>
                        @endif
                        </tbody>
                    </table>
                </div>
            </div>
            {!! make_page($page) !!}
        </div>
    </div>
@endsection

@section('scripts')
    <script src="{{ mix("plugins/User/js/user.js") }}"></script>
@endsection