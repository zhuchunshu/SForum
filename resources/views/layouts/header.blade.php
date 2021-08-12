@include('layouts.header_top')
<aside class="navbar navbar-vertical navbar-expand-lg navbar-dark">
    <div class="container-fluid">
        <button class="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbar-menu">
            <span class="navbar-toggler-icon"></span>
        </button>
        <h1 class="navbar-brand navbar-brand-autodark">
            <a href="/">
                {{ config('codefec.app.name', 'CodeFec') }}
            </a>
        </h1>
        <h1 class="navbar-brand navbar-brand-autodark">

        </h1>

        <div class="collapse navbar-collapse" id="navbar-menu">
            <ul class="navbar-nav pt-lg-3">
                @foreach (menu()->get() as $key => $value)
                    @if (!arr_has($value, 'parent_id'))
                        @if (!menu_pd($key))
                            @if ('/' . request()->path() == $value['url'])
                                <li class="nav-item active">
                                @else
                                <li class="nav-item">
                            @endif
                            <a class="nav-link" id="admin-menu-{{$key}}" href="{{ $value['url'] }}">
                                <span class="nav-link-icon d-md-none d-lg-inline-block">
                                    <!-- Download SVG icon from http://tabler-icons.io/i/package -->
                                    @if ($value['icon'])
                                        {!! $value['icon'] !!}
                                    @else
                                        <svg xmlns="http://www.w3.org/2000/svg"
                                            class="icon icon-tabler icon-tabler-circle" width="24" height="24"
                                            viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                                            stroke-linecap="round" stroke-linejoin="round">
                                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                            <circle cx="12" cy="12" r="9"></circle>
                                        </svg>
                                    @endif
                                </span>
                                <span class="nav-link-title">
                                    {{ $value['name'] }}
                                </span>
                            </a>
                                </li>
                        @else
                        @if (Helpers_Str()->is($value['url']."*","/".request()->path()))
                            <li class="nav-item active dropdown">
                            @else
                            <li class="nav-item dropdown">
                        @endif
                                <a class="nav-link dropdown-toggle" href="#navbar-base" data-bs-toggle="dropdown"
                                    role="button" aria-expanded="false">
                                    <span class="nav-link-icon d-md-none d-lg-inline-block">
                                        <!-- Download SVG icon from http://tabler-icons.io/i/package -->
                                        @if ($value['icon'])
                                            {!! $value['icon'] !!}
                                        @else
                                            <svg xmlns="http://www.w3.org/2000/svg"
                                                class="icon icon-tabler icon-tabler-circle" width="24" height="24"
                                                viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                                                stroke-linecap="round" stroke-linejoin="round">
                                                <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                                <circle cx="12" cy="12" r="9"></circle>
                                            </svg>
                                        @endif
                                    </span>
                                    <span class="nav-link-title">
                                        {{ $value['name'] }}
                                    </span>
                                </a>
                                <div class="dropdown-menu">
                                    <div class="dropdown-menu-columns">
                                        <div class="dropdown-menu-column">
                                            @foreach (menu_pdArr($key) as $keys => $values)
                                                @if ('/' . request()->path() == $values['url'])
                                                <a class="dropdown-item active" menu="active" id="admin-menu-{{$keys}}" href="{{ $values['url'] }}">
                                                @else
                                                <a class="dropdown-item" id="admin-menu-{{$keys}}" href="{{ $values['url'] }}">
                                                @endif
                                                    <span class="nav-link-icon d-md-none d-lg-inline-block">
                                                        <!-- Download SVG icon from http://tabler-icons.io/i/package -->
                                                        @if ($values['icon'])
                                                            {!! $values['icon'] !!}
                                                        @else
                                                            <svg xmlns="http://www.w3.org/2000/svg"
                                                                class="icon icon-tabler icon-tabler-circle" width="24"
                                                                height="24" viewBox="0 0 24 24" stroke-width="2"
                                                                stroke="currentColor" fill="none" stroke-linecap="round"
                                                                stroke-linejoin="round">
                                                                <path stroke="none" d="M0 0h24v24H0z" fill="none">
                                                                </path>
                                                                <circle cx="12" cy="12" r="9"></circle>
                                                            </svg>
                                                        @endif
                                                    </span>{{ $values['name'] }}
                                                </a>

                                            @endforeach
                                        </div>
                                    </div>
                                </div>
                            </li>
                        @endif
                    @endif
                @endforeach
            </ul>
        </div>
    </div>
</aside>
