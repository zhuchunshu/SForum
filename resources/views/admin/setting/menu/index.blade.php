@extends('app')
@section('title','菜单管理')
@section('content')

    <div class="row row-cards">
        <div class="col-12">
            <div class="card">
                <div class="card-header">
                    <h3 class="card-title">页头导航</h3>
                    <div class="card-actions">
                        <a href="/admin/setting/menu/create" >新增</a>
                        |
                        <a href="/admin/setting/menu/import" >导入(恢复)</a>
                    </div>
                </div>
                <div class="card-table table-responsive" id="vue-menu-list">
                    <table
                            class="table table-vcenter table-nowrap">
                        <thead>
                        <tr>
                            <th>#</th>
                            <th>名称</th>
                            <th>链接</th>
                            <th>icon</th>
                            <th>排序 <span class="text-muted">(数字越小越靠前)</span></th>
                            <th class="w-1"></th>
                        </tr>
                        </thead>
                        <tbody>
                        @if(!$page->count())
                            <td class="text-muted">无更多结果</td>
                            <td class="text-muted">无更多结果</td>
                            <td class="text-muted">无更多结果</td>
                            <td class="text-muted">无更多结果</td>
                            <td class="text-muted">无更多结果</td>
                            <td class="text-muted">无更多结果</td>
                        @else
                            @foreach($page as $menu)
                                <tr>
                                    <td>{{$menu['id']}}</td>
                                    <td>{{__($menu['name'])}}</td>
                                    <td><a href="{{$menu['url']}}">{{\Hyperf\Stringable\Str::limit($menu['url'],50)}}</a></td>
                                    <td>
                                        {!! $menu['icon'] !!}
                                    </td>
                                    <td>
                                        {!! $menu['sort'] !!}
                                    </td>
                                    <td><a href="/admin/setting/menu/{{$menu['id']}}/edit">修改</a> @if(!arr_has($menu,'Itf') || $menu['Itf']!==true) | <a href="#" @@click="del({{$menu['id']}})">删除</a> @endif</td>
                                </tr>
                            @endforeach
                        @endif
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    </div>

@endsection

@section('scripts')
    <script src="{{file_hash('js/admin/setting.js')}}"></script>
@endsection