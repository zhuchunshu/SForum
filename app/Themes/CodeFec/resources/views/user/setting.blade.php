@extends("App::app")
@section('title','个人设置')
@section('content')
    <div>
            <div class="row row-cards justify-content-center">
                <div class="col-md-12">
                    <div class="card">
                        <div class="card-header">
                            <ul class="nav nav-pills card-header-pills">
                                <li class="nav-item">
                                    <a class="nav-link @if(request()->input('m')!=='options') active @endif fw-bold" href="/user/setting">
                                        <svg xmlns="http://www.w3.org/2000/svg" class="icon me-1 d-none d-sm-block" width="24"
                                             height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                                             stroke-linecap="round" stroke-linejoin="round">
                                            <path d="M0 0h24v24H0z" stroke="none"/>
                                            <circle cx="12" cy="12" r="9"/>
                                            <path d="M12 7v5l3 3"/>
                                        </svg>
                                        基本设置</a>
                                </li>
                                <li class="nav-item">
                                    <a class="nav-link @if(request()->input('m')==='options') active @endif fw-bold" href="/user/setting?m=options">
                                        <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-settings" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                            <desc>Download more icon variants from https://tabler-icons.io/i/settings</desc>
                                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                            <path d="M10.325 4.317c.426 -1.756 2.924 -1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543 -.94 3.31 .826 2.37 2.37a1.724 1.724 0 0 0 1.065 2.572c1.756 .426 1.756 2.924 0 3.35a1.724 1.724 0 0 0 -1.066 2.573c.94 1.543 -.826 3.31 -2.37 2.37a1.724 1.724 0 0 0 -2.572 1.065c-.426 1.756 -2.924 1.756 -3.35 0a1.724 1.724 0 0 0 -2.573 -1.066c-1.543 .94 -3.31 -.826 -2.37 -2.37a1.724 1.724 0 0 0 -1.065 -2.572c-1.756 -.426 -1.756 -2.924 0 -3.35a1.724 1.724 0 0 0 1.066 -2.573c-.94 -1.543 .826 -3.31 2.37 -2.37c1 .608 2.296 .07 2.572 -1.065z"></path>
                                            <circle cx="12" cy="12" r="3"></circle>
                                        </svg>
                                        灵活设置</a>
                                </li>
                            </ul>
                        </div>
                        <div class="col-md-12">
                            @if(request()->input('m')!=='options')
                                @include('App::user.setting.common')
                            @else
                                @include('App::user.setting.options')
                            @endif
                        </div>
                    </div>
                </div>
            </div>
    </div>
@endsection

@section('scripts')
    <script src="{{mix("plugins/Core/js/user.js")}}"></script>
@endsection