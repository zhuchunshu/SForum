<footer class="footer footer-transparent d-print-none">
    <div class="container">
        <div class="flex-row-reverse text-center row align-items-center">
            <div class="col-lg-auto ms-lg-auto">
                {{-- <ul class="mb-0 list-inline list-inline-dots">
                    <li class="list-inline-item"><a href="./docs/index.html" class="link-secondary">Documentation</a></li>
                    <li class="list-inline-item"><a href="./license.html" class="link-secondary">License</a></li>
                    <li class="list-inline-item"><a href="https://github.com/tabler/tabler" target="_blank" class="link-secondary" rel="noopener">Source code</a></li>
                    <li class="list-inline-item">
                        <a href="https://github.com/sponsors/codecalm" target="_blank" class="link-secondary" rel="noopener">
                            <!-- Download SVG icon from http://tabler-icons.io/i/heart -->
                            <svg xmlns="http://www.w3.org/2000/svg" class="icon text-pink icon-filled icon-inline" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M19.5 13.572l-7.5 7.428l-7.5 -7.428m0 0a5 5 0 1 1 7.5 -6.566a5 5 0 1 1 7.5 6.572" /></svg>
                            Sponsor
                        </a>
                    </li>
                </ul> --}}
            </div>
            <div class="mt-3 col-12 col-lg-auto mt-lg-0">
                <ul class="mb-0 list-inline list-inline-dots">
                    <li class="list-inline-item">
                        Copyright &copy; {{ date('Y') }}
                        <a href="." class="link-secondary">{{ get_options('web_name', 'CodeFec') }}</a>.
{{--                        All rights reserved.--}}
                    </li>
                     @if(get_options("icp",null))
                        <li class="list-inline-item">
                            <a href="https://beian.miit.gov.cn" class="link-secondary" rel="noopener">{{get_options("icp")}}</a>
                        </li>
                    @endif
                    @if(get_options("ga_icp",null))
                        <li class="list-inline-item">
                            <img src="http://www.beian.gov.cn/img/new/gongan.png"  alt="gong an"/>
                            <a href="http://www.beian.gov.cn/portal/registerSystemInfo?recordcode={{get_num(get_options("ga_icp"))}}" class="link-secondary" rel="noopener">{{get_options("ga_icp")}}</a>
                        </li>
                    @endif
                </ul>
            </div>
        </div>
    </div>
</footer>
