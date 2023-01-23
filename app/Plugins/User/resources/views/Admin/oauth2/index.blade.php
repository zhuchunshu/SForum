@extends('app')
@section('title','第三方登陆')
@section('content')
    <div class="row row-cards">
        <div class="col-12">
            <form action="" method="POST">
                <x-csrf/>
                <div class="card">
                    <div class="card-header">
                        <h3 class="card-title">第三方登陆设置</h3>
                    </div>
                    <div class="card-body">
                        @if(count((new \App\Plugins\User\src\Service\Oauth2())->get_all_interface()))
                            <div class="card">
                                <div class="card-header">
                                    <ul class="nav nav-tabs card-header-tabs" data-bs-toggle="tabs">
                                        <li class="nav-item">
                                            <a href="#tabs-master" class="nav-link active" data-bs-toggle="tab"><!-- Download SVG icon from http://tabler-icons.io/i/home -->
                                                <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-aperture" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                                    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                                    <circle cx="12" cy="12" r="9"></circle>
                                                    <path d="M3.6 15h10.55"></path>
                                                    <path d="M6.551 4.938l3.26 10.034"></path>
                                                    <path d="M17.032 4.636l-8.535 6.201"></path>
                                                    <path d="M20.559 14.51l-8.535 -6.201"></path>
                                                    <path d="M12.257 20.916l3.261 -10.034"></path>
                                                </svg>主要</a>
                                        </li>
                                        @foreach((new \App\Plugins\User\src\Service\Oauth2())->get_all() as $data)
                                            <li class="nav-item">
                                                <a href="#tabs-interface-{{$data['mark']}}" class="nav-link" data-bs-toggle="tab"><!-- Download SVG icon from http://tabler-icons.io/i/user -->
                                                    {!! $data['icon'] !!}
                                                    {!! $data['name'] !!}
                                                </a>
                                            </li>
                                        @endforeach
                                    </ul>
                                </div>
                                <div class="card-body">
                                    <div class="tab-content">
                                        <div class="tab-pane active show" id="tabs-master">
                                            @include('User::Admin.oauth2.master')
                                        </div>
                                        @foreach((new \App\Plugins\User\src\Service\Oauth2())->get_all() as $data)
                                            <div class="tab-pane" id="tabs-interface-{{$data['mark']}}">
                                                @include($data['admin_view'])
                                            </div>
                                        @endforeach
                                    </div>
                                </div>
                            </div>
                        @else
                            <div class="empty">
                                <div class="empty-header">403</div>
                                <p class="empty-title">暂无接口</p>
                                <p class="empty-subtitle text-muted">
                                    暂无可用的第三方登陆接口，请尝试自己扩展或安装扩展插件
                                </p>
                            </div>
                        @endif
                    </div>
                    @if(count((new \App\Plugins\User\src\Service\Oauth2())->get_all_interface()))
                    <div class="card-footer">
                       <div class="row">
                           <div class="col"></div>
                           <div class="col-auto">
                               <button class="btn btn-primary">保存</button>
                           </div>
                       </div>
                    </div>
                    @endif
                </div>
            </form>
        </div>
    </div>
@endsection