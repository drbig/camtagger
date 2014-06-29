# camtagger 

Simple command-line mass-tagger for [camlistore](https://camlistore.org/).

Building camtagger requires camlistore packages, i.e. you should have proper 
dev setup already in place. This should be go-gettable.

For options and output format please read `camtagger -h`.

## Scenario

You have your camlistore mounted locally via 
[FUSE](http://fuse.sourceforge.net/). In your `/roots` you have one that you 
use for incremental backups (or just storage). When you first add files to that 
root it would be nice to tag them in bulk. This is where camtagger will help 
you.

For best experience you should use camtagger in conjunction with some decent 
file manager (like some [DOpus](http://en.wikipedia.org/wiki/DOpus) clone, e.g. 
[Worker](http://www.boomerangsworld.de/cms/worker/)), so that you can easily 
select a bunch of files and have a nice little pop-up window when you specify 
tags.

## Examples

    $ cd /mnt/camli/roots/Backup
    $ mkdir TaxDocs
    $ cp ~/tax*.pdf TaxDocs/
    $ camtagger add archive,important,doc -- TaxDocs/*
    /mnt/camli/root/Backup/TaxDocs/tax-2013.pdf +archive +important +doc
    /mnt/camli/root/Backup/TaxDocs/tax-2014.pdf +archive +important +doc
    ...

Time has moved on, we no longer pay taxes...

    $ camtagger del important -- /mnt/camli/root/Backup/TaxDocs/*
    /mnt/camli/root/Backup/TaxDocs/tax-2013.pdf -important
    /mnt/camli/root/Backup/TaxDocs/tax-2014.pdf -important
    ...

## Bugs

You may have noticed that the `-workers` parameter is set to `1` by default, 
this is because on my setup setting it to anything higher results in some 
weird deadlock on both ends (also, the web ui stops showing permanodes). If 
you're okay with such issue please feel free to try it with more workers. As of 
today I have no idea if the problem is in my code or the camlistore code. For 
reference my camlistore build is `2014-06-27-8a9ff86` with `go1.3` on `amd64` 
(both server and client).

## Copyright

Copyright (c) 2014 Piotr S. Staszewski

Absolutely no warranty. See LICENSE.txt for details.
