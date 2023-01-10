@extends("app")

@section('title','插件管理')
@section('headerBtn')
    <button class="btn btn-light" @@click="migrateAll" href="#">对所有已启动插件进行数据迁移</button>

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
                        <div class="text-muted">插件启停时网站出现短暂502为正常现象，因为SuperForum在优化/重建代理类，这个过程一般需要5秒左右</div>
                    </div>
                </div>
                <a class="btn-close" data-bs-dismiss="alert" aria-label="close"></a>
            </div>
        </div>
        @foreach (\App\CodeFec\Plugins::GetAll() as $key => $value)
            <div class="col-md-4">
                <div class="card">
                    @if(plugins()->getLogo($key))
                        <div class="card-stamp">
                            <div class="bg-yellow">
                                <!-- Download SVG icon from http://tabler-icons.io/i/bell -->
                                <img src="{{plugins()->getLogo($key)}}" alt="">
                            </div>
                        </div>
                    @endif
                    <div class="card-body">
                        <h3 class="card-title">
                            {{$value['data']['name']}}
                        </h3>
                        <p>
                            @if(@$value['data']['masterVersion'] && @$value['data']['masterVersion']>build_info()->version)
                                <span class="badge badge-outline text-orange">可能不兼容此插件,要求系统版本>={{$value['data']['masterVersion']}}</span>
                            @else
                                <span class="badge badge-outline text-green">当前插件兼容此程序!</span>
                            @endif
                        </p>
                        <p>插件目录: {{ '/app/Plugins/' . $key }}</p>
                        <p>插件作者: <a href="{{ $value['data']['link'] }}">{{ $value['data']['author'] }}</a></p>
                        <p>插件版本:{{ $value['data']['version'] }}</p>
                        <p>插件描述:{{ $value['data']['description'] }}</p>
                    </div>
                    <div class="card-footer">
                        <div class="row align-items-center">
                            <div class="col-auto">
                                <a @@click="migrate('{{ $value['dir'] }}','{{ $value['path'] }}')" href="#">数据迁移</a>
                                |
                                <a @@click="remove('{{ $value['dir'] }}','{{ $value['path'] }}')" href="#">卸载</a>
                            </div>
                            <div class="col-auto ms-auto">
                                <label class="form-check form-switch">
                                    <input class="form-check-input" value="{{ $value['dir'] }}" type="checkbox"
                                           v-model="switchs">
                                </label>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        @endforeach
    </div>

@endsection
