@extends('app')
@section('title','导入页头菜单')
@section('content')
    <div class="row row-cards">
        <div class="col-12">
            <div class="card">
                <div class="card-header">
                    <h3 class="card-title">导入页头菜单</h3>
                </div>
                <div class="card-table table-responsive" id="vue-setting-menu-import">
                    <table class="table card-table table-vcenter">
                        <thead>
                        <tr>
                            <th>文件名</th>
                            <th>创建时间</th>
                            <th></th>
                            <th></th>
                            <th></th>
                        </tr>
                        </thead>
                        <tbody>
                        @if(!$page->count())
                            <tr>
                                <td class="text-muted">暂无更多结果</td>
                                <td class="text-muted">暂无更多结果</td>
                                <td class="text-muted">暂无更多结果</td>
                                <td class="text-muted">暂无更多结果</td>
                                <td class="text-muted">暂无更多结果</td>
                            </tr>
                        @else
                            @foreach($page as $component)
                                <tr>
                                    <td class="text-muted">
                                        {{$component['filename']}}
                                    </td>
                                    <td class="text-muted">
                                        {{$component['created_at']}}
                                    </td>
                                    <td class="text-muted w-5">
                                        <a @@click="recover('{{$component['path']}}')" href="#">恢复</a>
                                    </td>
                                    <td class="text-muted w-5">
                                        <a @@click="im('{{$component['path']}}')" href="#">导入</a>
                                    </td>
                                    <td class="text-muted w-5">
                                        <a @@click="rm('{{$component['path']}}')" href="#">删除</a>
                                    </td>
                                </tr>
                            @endforeach
                        @endif
                        </tbody>
                    </table>
                    <div class="mt-3">
                        {!! make_page($page) !!}
                    </div>
                </div>
                <div class="card-footer">
                    没找到备份文件? 可以手动上传到 <span class="text-red">{{BASE_PATH."/runtime/backup/menu"}}</span> 目录下
                </div>
            </div>
        </div>
    </div>
@endsection
@section('headerBtn')
    <a href="/admin/setting/menu" class="btn btn-primary">列表</a>
@endsection
@section('scripts')
    <script src="{{file_hash('js/admin/setting.js')}}"></script>
@endsection