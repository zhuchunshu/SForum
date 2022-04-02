@extends("app")

@section('title','主题管理')
@section('headerBtn')
    <button class="btn btn-light" @@click="migrateAll" href="#">对已启动主题进行数据迁移</button>
@endsection

@section('pageId','vue-theme-table')

@section('content')
    <div class="row row-cards">
        @foreach (theme()->get() as $key => $value)
            <div class="col-md-4">
                <div class="card">
                    @if(theme()->getLogo($key))
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
                        <p>主题目录: {{ '/app/Plugins/' . $key }}</p>
                        <p>主题作者: <a href="{{ $value['data']['link'] }}">{{ $value['data']['author'] }}</a></p>
                        <p>主题版本:{{ $value['data']['version'] }}</p>
                        <p>主题描述:{{ $value['data']['description'] }}</p>
                    </div>
                    <div class="card-footer">
                        <div class="row align-items-center">
                            <div class="col-auto">
                                <a @@click="migrate('{{ $value['dir'] }}','{{ $value['path'] }}')" href="#">数据迁移</a>
                                |
                                <a @@click="remove('{{ $value['dir'] }}','{{ $value['path'] }}')" href="#">卸载</a>
                            </div>
                            <div class="col-auto ms-auto">
                                <a @@click="Setenable('{{ $value['dir'] }}')" href="#" @if (get_options("theme",'CodeFec')===$value['dir']) class="btn btn-sm btn-outline-info disabled" @else class="btn btn-sm btn-outline-info" @endif>启用</a>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        @endforeach
    </div>

@endsection
