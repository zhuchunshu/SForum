<?php

declare(strict_types=1);

namespace App\Command\CodeFec;

use Hyperf\Command\Command as HyperfCommand;
use Hyperf\Command\Annotation\Command;
use Hyperf\Database\Migrations\Migrator;
use Psr\Container\ContainerInterface;
use Symfony\Component\Console\Input\InputArgument;
use Symfony\Component\Console\Input\InputOption;

/**
 * @Command
 */
class migrate extends HyperfCommand
{
    /**
     * @var ContainerInterface
     */
    protected $container;

    /**
     * The migrator instance.
     *
     * @var Migrator
     */
    protected $migrator;

    public function __construct(ContainerInterface $container,Migrator $migrator)
    {
        $this->container = $container;


        parent::__construct('CodeFec:migrate');
        $this->migrator = $migrator;
    }

    public function configure()
    {
        parent::configure();
        $this->setDescription('CodeFec Database Migrate');
    }

    public function handle()
    {
        $path = $this->input->getArgument('path') ?? BASE_PATH."/migrations";
        $arr = array_merge(
            $this->migrator->paths(),
            [$path]
        );
        $this->migrator->setOutput($this->output)
            ->run($path,[
                'pretend' => $this->input->getOption('pretend'),
                'step' => $this->input->getOption('step'),
            ]);
        $this->info("success, path:".$path);
    }
    protected function getArguments(): array
    {
        return [
            ['path', InputArgument::OPTIONAL, 'migrations路径']
        ];
    }
    protected function getOptions(): array
    {
        return [
            ['database', null, InputOption::VALUE_OPTIONAL, 'The database connection to use'],
            ['force', null, InputOption::VALUE_NONE, 'Force the operation to run when in production'],
            ['path', null, InputOption::VALUE_OPTIONAL, 'The path to the migrations files to be executed'],
            ['realpath', null, InputOption::VALUE_NONE, 'Indicate any provided migration file paths are pre-resolved absolute paths'],
            ['pretend', null, InputOption::VALUE_NONE, 'Dump the SQL queries that would be run'],
            ['seed', null, InputOption::VALUE_NONE, 'Indicates if the seed task should be re-run'],
            ['step', null, InputOption::VALUE_NONE, 'Force the migrations to be run so they can be rolled back individually'],
        ];
    }
}
