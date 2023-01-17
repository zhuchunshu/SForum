@extends("app")

@section('title','插件管理')
@section('headerBtn')
    <button class="btn btn-light" @@click="migrateAll" href="#">所有插件数据迁移</button>

    <button class="btn btn-dark" @@click="updatePluginsPackage" style="margin-left: 5px">更新插件包</button>
@endsection

@section('pageId','vue-plugin-table')

@section('content')
    <div class="row row-cards">
        <div class="col-lg-12">
            <div class="alert alert-info alert-dismissible" role="alert">
                <div class="d-flex">
                    <div>
                        <!-- Download SVG icon from http://tabler-icons.io/i/info-circle -->
                        <svg xmlns="http://www.w3.org/2000/svg" class="icon alert-icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><circle cx="12" cy="12" r="9" /><line x1="12" y1="8" x2="12.01" y2="8" /><polyline points="11 12 12 12 12 16 13 16" /></svg>
                    </div>
                    <div>
                        <h4 class="alert-title">Did you know?</h4>
                        <div class="text-muted">SForum已废弃插件启停功能，不用的插件建议直接卸载。</div>
                    </div>
                </div>
                <a class="btn-close" data-bs-dismiss="alert" aria-label="close"></a>
            </div>
        </div>
        <div class="col-lg-12">
            <div class="card">
                <div class="card-header"><h3 class="card-title">插件列表</h3></div>
                <div class="card-table table-responsive">
                    <table
                            class="table table-vcenter table-nowrap">
                        <thead>
                        <tr>
                            <th>插件名</th>
                            <th>兼容性</th>
                            <th>作者</th>
                            <th>版本</th>
                            <th>描述</th>
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
                            @foreach($page as $plugin)
                                <tr>
                                    <td class="text-center">
                                        @if(plugins()->has_logo($plugin['dir']))
                                            <span class="avatar avatar-md"><img src="/admin/plugins/logo?plugin={{$plugin['dir']}}" alt=""></span>
                                        @endif <div>{{$plugin['data']['name']}}</div></td>
                                    <td class="text-muted" >
                                        @if(@$plugin['data']['masterVersion'] && @$plugin['data']['masterVersion']>build_info()->version)
                                            <span class="status status-red">不兼容,要求SForum版本>={{$plugin['data']['masterVersion']}}</span>
                                        @else
                                            <span class="status status-green">兼容!</span>
                                        @endif
                                    </td>
                                    <td class="text-muted" ><a href="{{ $plugin['data']['link'] }}">{{ $plugin['data']['author'] }}</a></td>
                                    <td class="text-muted" >
                                        {{$plugin['data']['version']}}
                                    </td>
                                    <td>
                                        {!! $plugin['data']['description'] !!}
                                    </td>
                                    <td>
                                        <a @@click="migrate('{{ $plugin['dir'] }}','{{ $plugin['path'] }}')" href="#">数据迁移</a>
                                        |
                                        <a @@click="remove('{{ $plugin['dir'] }}','{{ $plugin['path'] }}')" href="#">卸载</a>
                                    </td>
                                </tr>
                            @endforeach
                        @endif
                        </tbody>
                    </table>
                </div>
                @if($page->count() && (int)request()->input('page',1)!==1)
                    <div class="card-footer">
                        {!! make_page($page) !!}
                    </div>
                @endif
            </div>
        </div>
    </div>

@endsection
